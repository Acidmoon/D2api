package securityaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	ErrNoPromptText = errors.New("prompt audit request contains no user text")

	bearerPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+\-/]+=*`)
	apiKeyPattern = regexp.MustCompile(`(?i)\b(sk|rk|pk|api[_-]?key|token|secret|password)[-_:=\s]+[A-Za-z0-9._~+\-/]{8,}`)
	canaryPattern = regexp.MustCompile(`(?i)([A-Z]+_CANARY_)[A-Za-z0-9_-]+`)
	emailPattern  = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	phonePattern  = regexp.MustCompile(`(?:\+?\d[\d\s().-]{8,}\d)`)
)

const promptAuditPrioritySeparator = "\x00SUB2API_PROMPT_AUDIT_PRIORITY_END\x00"

type promptSegment struct {
	text string
	user bool
	role string
}

// ExtractPromptSnapshot builds the complete audit snapshot for asynchronous
// auditing. asyncLatestTurnOnly narrows extraction to the single latest user
// message (matching legacy content_moderation's last-user-message collection);
// auditRoles restricts extraction to the listed message roles; a nil or empty
// list defaults to user-only extraction (see DefaultAuditRoles).
func ExtractPromptSnapshot(req Request, asyncLatestTurnOnly bool, auditRoles []string) (PromptSnapshot, error) {
	return extractPromptSnapshot(req, false, asyncLatestTurnOnly, auditRoles)
}

// ExtractBlockingPromptSnapshot builds the narrow, low-latency blocking input
// when configured. Asynchronous auditing likewise applies audit_roles: only
// configured-role messages are retained rather than the complete client-
// controlled transcript. Blocking behavior is deliberately independent of the
// async latest-turn-only setting (BlockingLatestTurnOnly governs it instead).
func ExtractBlockingPromptSnapshot(req Request, latestTurnOnly bool, auditRoles []string) (PromptSnapshot, error) {
	return extractPromptSnapshot(req, latestTurnOnly, false, auditRoles)
}

func extractPromptSnapshot(req Request, latestTurnOnly bool, asyncLatestTurnOnly bool, auditRoles []string) (PromptSnapshot, error) {
	var document any
	if err := json.Unmarshal(req.Body, &document); err != nil {
		return PromptSnapshot{}, errors.New("prompt audit request JSON is invalid")
	}
	roles := effectiveAuditRoles(auditRoles)
	var segments []string
	switch {
	case latestTurnOnly:
		// Blocking 收窄必须先于角色过滤：在全量（未过滤）段上先做
		// blockingSegmentsLatestUserAndPreviousOutput 收窄，再对收窄结果按角色过滤。
		// 若先过滤再收窄，默认 user-only 下历史 user 段会因彼此相邻而合并成一段，
		// 前一轮 assistant/model 输出也永远进不了收窄结果。
		unfiltered := extractProtocolSegments(req.Protocol, document, AuditRoleNames)
		narrowed := blockingSegmentsLatestUserAndPreviousOutput(unfiltered)
		segments = promptSegmentTexts(filterSegmentsByAuditRoles(narrowed, roles))
	case asyncLatestTurnOnly:
		// Async 收窄同样定位原始消息结构里的「最后一条 user 消息」而非过滤后拼接：
		// 角色过滤会抹掉消息边界（如 [user,assistant,user] 在 user-only 下变成相邻
		// 两段），只有回到原始 messages/input/contents 数组才能精确定位最后一条
		// user 消息（与 blocking 收窄先于角色过滤同理）。收窄结果仍按 audit_roles
		// 过滤：若角色不含 user，结果为 ErrNoPromptText。
		narrowed := extractLatestUserTurnSegments(req.Protocol, document)
		segments = normalizeSegmentsLatestUserFirst(filterSegmentsByAuditRoles(narrowed, roles))
	default:
		extracted := extractProtocolSegments(req.Protocol, document, roles)
		segments = normalizeSegmentsLatestUserFirst(extracted)
	}
	if len(segments) == 0 {
		return PromptSnapshot{}, ErrNoPromptText
	}
	scanText, metadataText := buildPrioritizedScanText(segments)
	digest := sha256.Sum256([]byte(metadataText))
	stage := strings.TrimSpace(req.Stage)
	if stage == "" {
		stage = "http"
	}
	return PromptSnapshot{
		RequestID: req.RequestID, UserID: req.UserID, UsernameSnapshot: req.Username,
		UserEmailSnapshot: req.UserEmail, APIKeyID: req.APIKeyID, APIKeyNameSnapshot: req.APIKeyName,
		GroupID: cloneInt64Ptr(req.GroupID), GroupName: req.GroupName, Provider: req.Provider,
		Endpoint: req.Endpoint, Protocol: req.Protocol, Model: req.Model,
		PromptHash: hex.EncodeToString(digest[:]), RedactedPreview: BuildPromptPreview(metadataText, DefaultPromptPreviewMaxRunes),
		FullPrompt:   BuildFullPrompt(metadataText, DefaultFullPromptMaxRunes),
		PromptLength: utf8.RuneCountInString(metadataText), MessageCount: len(segments), Stage: stage,
		ScanText: scanText,
	}, nil
}

// DefaultPromptPreviewMaxRunes caps how much sanitized prompt text may be
// considered before BuildPromptPreview withholds the majority for storage/UI.
const DefaultPromptPreviewMaxRunes = 96

// DefaultFullPromptMaxRunes caps how much unredacted prompt text is persisted
// on an audit event for admin review. It is deliberately generous so realistic
// prompts are kept intact while bounding per-row storage.
const DefaultFullPromptMaxRunes = 65536

func extractProtocolSegments(protocol string, document any, auditRoles []string) []promptSegment {
	root, _ := document.(map[string]any)
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	switch protocol {
	case "openai_chat_completions", "openai_chat", "chat_completions":
		return extractChatLikeSegments(root, auditRoles)
	case "anthropic_messages", "claude_messages", "messages":
		return append(extractAnthropicSystem(root["system"], auditRoles), extractMessages(root["messages"], auditRoles)...)
	case "gemini", "gemini_generate_content":
		return extractGeminiRoot(root, auditRoles)
	case "openai_responses", "responses", "responses_websocket":
		if frameType := stringValue(root["type"]); frameType != "" || protocol == "responses_websocket" {
			if frameType != "response.create" {
				return nil
			}
			if input, exists := root["input"]; exists && input != nil {
				return append(extractInstructions(root["instructions"], auditRoles), extractResponses(input, auditRoles)...)
			}
			if response, ok := root["response"].(map[string]any); ok {
				return append(extractInstructions(response["instructions"], auditRoles), extractResponses(response["input"], auditRoles)...)
			}
			return extractInstructions(root["instructions"], auditRoles)
		}
		return append(extractInstructions(root["instructions"], auditRoles), extractResponses(root["input"], auditRoles)...)
	case "openai_images", "grok_media", "media", "images":
		return mediaPromptSegments(root, auditRoles)
	default:
		if segments := extractChatLikeSegments(root, auditRoles); len(segments) > 0 {
			return segments
		}
		if responses := append(extractInstructions(root["instructions"], auditRoles), extractResponses(root["input"], auditRoles)...); len(responses) > 0 {
			return responses
		}
		if gemini := extractGeminiRoot(root, auditRoles); len(gemini) > 0 {
			return gemini
		}
		return mediaPromptSegments(root, auditRoles)
	}
}

// extractLatestUserTurnSegments 按协议从原始请求结构中定位并提取「最后一条
// user 消息」的文本段（AsyncLatestTurnOnly 收窄），对齐 legacy content_moderation
// 的 collectLastRoleMessage / collectLastAnthropicUserMessage /
// collectLastResponsesInput / collectLastGeminiContent 收集语义，并扩展为从尾部
// 向前扫描定位最后一条 user 消息（legacy 只取数组最后一条且要求其为 user，这里是
// 超集）：定位的是 messages/input/contents 数组中最后一条 user 消息，而不是对整段
// 历史按角色过滤后拼接——过滤会抹掉消息边界，相邻 user 消息会混成一段。返回段保留
// role 信息，供调用方继续做 audit_roles 过滤与优先段标记。媒体类请求没有消息数组
// 结构，整个请求即一次用户提示词，无可收窄的历史，保持全量确定性文本提示词。
func extractLatestUserTurnSegments(protocol string, document any) []promptSegment {
	root, _ := document.(map[string]any)
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	switch protocol {
	case "openai_chat_completions", "openai_chat", "chat_completions",
		"anthropic_messages", "claude_messages", "messages":
		return lastUserMessageSegments(root["messages"])
	case "gemini", "gemini_generate_content":
		return lastGeminiUserTurnSegments(root)
	case "openai_responses", "responses", "responses_websocket":
		if frameType := stringValue(root["type"]); frameType != "" || protocol == "responses_websocket" {
			if frameType != "response.create" {
				return nil
			}
			if input, exists := root["input"]; exists && input != nil {
				return lastResponsesUserMessageSegments(input)
			}
			if response, ok := root["response"].(map[string]any); ok {
				return lastResponsesUserMessageSegments(response["input"])
			}
			return nil
		}
		return lastResponsesUserMessageSegments(root["input"])
	case "openai_images", "grok_media", "media", "images":
		return mediaPromptSegments(root, AuditRoleNames)
	default:
		if segments := lastUserMessageSegments(root["messages"]); len(segments) > 0 {
			return segments
		}
		if segments := lastResponsesUserMessageSegments(root["input"]); len(segments) > 0 {
			return segments
		}
		if segments := lastGeminiUserTurnSegments(root); len(segments) > 0 {
			return segments
		}
		return mediaPromptSegments(root, AuditRoleNames)
	}
}

// lastUserMessageSegments 从 messages 数组中从尾部向前定位最后一条 user 消息，
// 提取其全部文本段（同一消息的多文本块全部保留），对齐 legacy collectLastRoleMessage
// 收集整条 user 消息的语义并扩展为向后扫描定位（legacy 只认数组最后一条且必须是
// user，这里允许其前面存在非 user 消息）。找不到 user 消息时返回 nil。
func lastUserMessageSegments(value any) []promptSegment {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	for index := len(items) - 1; index >= 0; index-- {
		message, ok := items[index].(map[string]any)
		if !ok {
			continue
		}
		if strings.ToLower(stringValue(message["role"])) != "user" {
			continue
		}
		result := make([]promptSegment, 0, 2)
		for _, text := range contentTexts(message["content"]) {
			result = append(result, promptSegment{text: text, user: true, role: "user"})
		}
		return result
	}
	return nil
}

// lastResponsesUserMessageSegments 定位 responses input 中最后一条 user 输入
// （role 为空或 user 的条目；纯字符串 input 视为一条 user 消息），对齐 legacy
// collectLastResponsesInput 的 isResponsesUserTextItem 判定。
func lastResponsesUserMessageSegments(value any) []promptSegment {
	if items, ok := value.([]any); ok {
		for index := len(items) - 1; index >= 0; index-- {
			if !isResponsesUserItem(items[index]) {
				continue
			}
			// 包成单元素数组复用 extractResponses 的数组分支：它同时处理 content 数组
			// 与顶层 text（input_text 条目），而单 map 分支只认 content。
			if segments := extractResponses([]any{items[index]}, AuditRoleNames); len(segments) > 0 {
				return normalizeNarrowedSegmentRoles(segments)
			}
		}
		return nil
	}
	if !isResponsesUserItem(value) {
		return nil
	}
	return normalizeNarrowedSegmentRoles(extractResponses([]any{value}, AuditRoleNames))
}

// normalizeNarrowedSegmentRoles 将 role 为空（input_text/无角色条目等）的段标记
// 为 user：后续 audit_roles 过滤按精确 role 匹配，空 role 会被误过滤，而空 role
// 条目在 responses/gemini 语义中即用户输入（与 isResponsesUserItem 判定一致）。
func normalizeNarrowedSegmentRoles(segments []promptSegment) []promptSegment {
	for index := range segments {
		if segments[index].role == "" {
			segments[index].user = true
			segments[index].role = "user"
		}
	}
	return segments
}

// isResponsesUserItem 判定 responses 条目是否算 user 消息：纯字符串输入、
// role 为空（input_text 等）或显式 role=user。
func isResponsesUserItem(value any) bool {
	switch typed := value.(type) {
	case string:
		return true
	case map[string]any:
		role := strings.ToLower(stringValue(typed["role"]))
		return role == "" || role == "user"
	default:
		return false
	}
}

// lastGeminiUserTurnSegments 从 contents/content/instances（及 requests 内嵌套的
// 同名字段）中定位最后一条 user 内容并提取其文本段，对齐 legacy
// collectLastGeminiContent 的 role 为空或 user 判定。源顺序与 extractGeminiRoot
// 一致（根字段在前，requests 按正序追加），随后从尾部向前扫描取最后命中的一条，
// 保证文档序「最后」= 最后一个 request 内的 user 内容。
func lastGeminiUserTurnSegments(root map[string]any) []promptSegment {
	if root == nil {
		return nil
	}
	sources := []any{root["contents"], root["content"], root["instances"]}
	if requests, ok := root["requests"].([]any); ok {
		for _, item := range requests {
			if request, ok := item.(map[string]any); ok {
				sources = append(sources, request["contents"], request["content"], request["instances"])
			}
		}
	}
	for index := len(sources) - 1; index >= 0; index-- {
		if segments := lastGeminiUserItemSegments(sources[index]); len(segments) > 0 {
			return segments
		}
	}
	return nil
}

// lastGeminiUserItemSegments 从单个 gemini 源（contents 数组、instances 数组或
// 单条内容对象）中定位最后一条 user 消息。instances[].prompt 视为 user 提示词。
func lastGeminiUserItemSegments(value any) []promptSegment {
	switch typed := value.(type) {
	case []any:
		for index := len(typed) - 1; index >= 0; index-- {
			item, ok := typed[index].(map[string]any)
			if !ok {
				continue
			}
			if prompt := stringValue(item["prompt"]); prompt != "" {
				return []promptSegment{{text: prompt, user: true, role: "user"}}
			}
			role := strings.ToLower(stringValue(item["role"]))
			if role != "" && role != "user" {
				continue
			}
			if segments := extractGemini(item, AuditRoleNames); len(segments) > 0 {
				return normalizeNarrowedSegmentRoles(segments)
			}
		}
		return nil
	case map[string]any:
		if prompt := stringValue(typed["prompt"]); prompt != "" {
			return []promptSegment{{text: prompt, user: true, role: "user"}}
		}
		role := strings.ToLower(stringValue(typed["role"]))
		if role != "" && role != "user" {
			return nil
		}
		return normalizeNarrowedSegmentRoles(extractGemini(typed, AuditRoleNames))
	default:
		return nil
	}
}

// effectiveAuditRoles 返回实际生效的审计提取角色：nil/空列表按默认
// DefaultAuditRoles（仅 user）处理，并归一化为小写、去重（保持传入顺序，
// 提取层允许任意角色名，配置层另有合法性校验）。
func effectiveAuditRoles(roles []string) []string {
	if len(roles) == 0 {
		return append([]string(nil), DefaultAuditRoles...)
	}
	seen := make(map[string]struct{}, len(roles))
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	if len(result) == 0 {
		return append([]string(nil), DefaultAuditRoles...)
	}
	return result
}

func rolesContain(roles []string, role string) bool {
	target := strings.ToLower(strings.TrimSpace(role))
	for _, candidate := range roles {
		if candidate == target {
			return true
		}
	}
	return false
}

func extractChatLikeSegments(root map[string]any, auditRoles []string) []promptSegment {
	if root == nil {
		return nil
	}
	return extractMessages(root["messages"], auditRoles)
}

func extractMessages(value any, auditRoles []string) []promptSegment {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	wanted := make(map[string]struct{}, len(auditRoles))
	for _, role := range auditRoles {
		wanted[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
	}
	result := make([]promptSegment, 0, len(items))
	for _, item := range items {
		message, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := strings.ToLower(stringValue(message["role"]))
		if _, match := wanted[role]; !match {
			continue
		}
		texts := contentTexts(message["content"])
		for _, text := range texts {
			result = append(result, promptSegment{text: text, user: role == "user", role: role})
		}
	}
	return result
}

func extractInstructions(value any, auditRoles []string) []promptSegment {
	if !rolesContain(auditRoles, "system") {
		return nil
	}
	switch typed := value.(type) {
	case string:
		if text := strings.TrimSpace(typed); text != "" {
			return []promptSegment{{text: text, role: "system"}}
		}
	case []any:
		return systemPromptSegments(contentTexts(typed))
	case map[string]any:
		return systemPromptSegments(contentTexts(typed))
	}
	return nil
}

func extractAnthropicSystem(value any, auditRoles []string) []promptSegment {
	if !rolesContain(auditRoles, "system") {
		return nil
	}
	switch typed := value.(type) {
	case string:
		if text := strings.TrimSpace(typed); text != "" {
			return []promptSegment{{text: text, role: "system"}}
		}
	case []any:
		return systemPromptSegments(contentTexts(typed))
	case map[string]any:
		return systemPromptSegments(contentTexts(typed))
	}
	return nil
}

func extractResponses(value any, auditRoles []string) []promptSegment {
	switch typed := value.(type) {
	case string:
		if !rolesContain(auditRoles, "user") {
			return nil
		}
		return []promptSegment{{text: typed, user: true, role: "user"}}
	case []any:
		result := make([]promptSegment, 0, len(typed))
		for _, item := range typed {
			switch entry := item.(type) {
			case string:
				if !rolesContain(auditRoles, "user") {
					continue
				}
				result = append(result, promptSegment{text: entry, user: true, role: "user"})
			case map[string]any:
				role := strings.ToLower(stringValue(entry["role"]))
				if role == "" {
					if !rolesContain(auditRoles, "user") {
						continue
					}
				} else if !rolesContain(auditRoles, role) {
					continue
				}
				if content, exists := entry["content"]; exists {
					for _, text := range contentTexts(content) {
						result = append(result, promptSegment{text: text, user: role == "" || role == "user", role: role})
					}
				} else if text := stringValue(entry["text"]); text != "" {
					result = append(result, promptSegment{text: text, user: role == "" || role == "user", role: role})
				}
			}
		}
		return result
	case map[string]any:
		role := strings.ToLower(stringValue(typed["role"]))
		if role == "" {
			if !rolesContain(auditRoles, "user") {
				return nil
			}
		} else if !rolesContain(auditRoles, role) {
			return nil
		}
		return promptSegmentsForRole(contentTexts(typed["content"]), role)
	default:
		return nil
	}
}

func extractGemini(value any, auditRoles []string) []promptSegment {
	var contents []any
	switch typed := value.(type) {
	case []any:
		contents = typed
	case map[string]any:
		contents = []any{typed}
	default:
		return nil
	}
	result := make([]promptSegment, 0, len(contents))
	for _, item := range contents {
		content, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := strings.ToLower(stringValue(content["role"]))
		if role == "" {
			if !rolesContain(auditRoles, "user") {
				continue
			}
		} else if !rolesContain(auditRoles, role) {
			continue
		}
		parts, _ := content["parts"].([]any)
		for _, part := range parts {
			if object, ok := part.(map[string]any); ok {
				if text := stringValue(object["text"]); text != "" {
					result = append(result, promptSegment{text: text, user: role == "" || role == "user", role: role})
				}
			}
		}
	}
	return result
}

func extractGeminiRoot(root map[string]any, auditRoles []string) []promptSegment {
	if root == nil {
		return nil
	}
	result := extractGeminiSystemInstruction(root["systemInstruction"], auditRoles)
	result = append(result, extractGeminiSystemInstruction(root["system_instruction"], auditRoles)...)
	result = append(result, extractGemini(root["contents"], auditRoles)...)
	result = append(result, extractGemini(root["content"], auditRoles)...)
	result = append(result, extractGeminiInstances(root["instances"], auditRoles)...)
	if requests, ok := root["requests"].([]any); ok {
		for _, item := range requests {
			request, ok := item.(map[string]any)
			if !ok {
				continue
			}
			result = append(result, extractGeminiSystemInstruction(request["systemInstruction"], auditRoles)...)
			result = append(result, extractGeminiSystemInstruction(request["system_instruction"], auditRoles)...)
			result = append(result, extractGemini(request["contents"], auditRoles)...)
			result = append(result, extractGemini(request["content"], auditRoles)...)
			result = append(result, extractGeminiInstances(request["instances"], auditRoles)...)
		}
	}
	return result
}

func extractGeminiSystemInstruction(value any, auditRoles []string) []promptSegment {
	if !rolesContain(auditRoles, "system") {
		return nil
	}
	switch typed := value.(type) {
	case string:
		if text := strings.TrimSpace(typed); text != "" {
			return []promptSegment{{text: text, role: "system"}}
		}
	case map[string]any:
		if parts, ok := typed["parts"].([]any); ok {
			result := make([]promptSegment, 0, len(parts))
			for _, part := range parts {
				if object, ok := part.(map[string]any); ok {
					if text := stringValue(object["text"]); text != "" {
						result = append(result, promptSegment{text: text, role: "system"})
					}
				}
			}
			return result
		}
		return systemPromptSegments(contentTexts(typed))
	case []any:
		segments := extractGemini(typed, auditRoles)
		for index := range segments {
			segments[index].user = false
			segments[index].role = "system"
		}
		return segments
	}
	return nil
}

func extractGeminiInstances(value any, auditRoles []string) []promptSegment {
	if !rolesContain(auditRoles, "user") {
		return nil
	}
	instances, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]promptSegment, 0, len(instances))
	for _, item := range instances {
		if instance, ok := item.(map[string]any); ok {
			if prompt := stringValue(instance["prompt"]); prompt != "" {
				result = append(result, promptSegment{text: prompt, user: true, role: "user"})
			}
		}
	}
	return result
}

// mediaPromptSegments 将图片/媒体类请求的确定性文本提示词视为 user 角色段。
func mediaPromptSegments(root map[string]any, auditRoles []string) []promptSegment {
	if !rolesContain(auditRoles, "user") {
		return nil
	}
	return userPromptSegments(extractMediaPrompts(root))
}

func extractMediaPrompts(root map[string]any) []string {
	if root == nil {
		return nil
	}
	result := make([]string, 0, 4)
	seen := map[string]struct{}{}
	var walk func(any, string)
	walk = func(value any, key string) {
		switch typed := value.(type) {
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for childKey := range typed {
				keys = append(keys, childKey)
			}
			sort.Strings(keys)
			for _, childKey := range keys {
				walk(typed[childKey], childKey)
			}
		case []any:
			for _, item := range typed {
				walk(item, key)
			}
		case string:
			if !isMediaPromptKey(key) || looksLikeMediaPayload(typed) {
				return
			}
			text := strings.TrimSpace(typed)
			if text == "" {
				return
			}
			if _, duplicate := seen[text]; duplicate {
				return
			}
			seen[text] = struct{}{}
			result = append(result, text)
		}
	}
	walk(root, "")
	return result
}

func isMediaPromptKey(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	switch normalized {
	case "prompt", "inputprompt", "textprompt", "description", "query", "lyrics", "negativeprompt",
		"positiveprompt", "gptdescriptionprompt", "prompten", "finalprompt", "finalzhprompt",
		"origprompt", "actualprompt", "imageprompt", "input":
		return true
	default:
		return false
	}
}

func looksLikeMediaPayload(value string) bool {
	trimmed := strings.TrimSpace(value)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "data:image/") || strings.HasPrefix(lower, "data:video/") ||
		strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}
	if len(trimmed) >= 256 {
		for _, r := range trimmed {
			alphaNumeric := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
			if !alphaNumeric && r != '+' && r != '/' && r != '=' {
				return false
			}
		}
		return true
	}
	return false
}

func contentTexts(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []any:
		result := make([]string, 0, len(typed))
		for _, part := range typed {
			object, ok := part.(map[string]any)
			if !ok {
				continue
			}
			typeName := strings.ToLower(stringValue(object["type"]))
			if typeName != "" && typeName != "text" && typeName != "input_text" {
				continue
			}
			if text := stringValue(object["text"]); text != "" {
				result = append(result, text)
			}
		}
		return result
	case map[string]any:
		if text := stringValue(typed["text"]); text != "" {
			return []string{text}
		}
	}
	return nil
}

func normalizeSegmentsLatestUserFirst(values []promptSegment) []string {
	normalized := normalizedPromptSegments(values)
	if len(normalized) == 0 {
		return nil
	}
	priorityIndex := len(normalized) - 1
	for index := len(normalized) - 1; index >= 0; index-- {
		if isUserSegment(normalized[index]) {
			priorityIndex = index
			break
		}
	}
	result := make([]string, 0, len(normalized))
	result = append(result, normalized[priorityIndex].text)
	for index, segment := range normalized {
		if index != priorityIndex {
			result = append(result, segment.text)
		}
	}
	return result
}

// blockingSegmentsLatestUserAndPreviousOutput limits synchronous guard input to
// the current user turn and the nearest preceding assistant/model turn. It is
// deliberately opt-in because full transcript scanning remains stronger at
// finding client-controlled content placed in older or non-user messages.
// The narrowed segments keep their role info so callers can apply audit-role
// filtering afterwards.
func blockingSegmentsLatestUserAndPreviousOutput(values []promptSegment) []promptSegment {
	normalized := normalizedPromptSegments(values)
	latestUserStart := latestUserSegmentStart(normalized)
	if latestUserStart < 0 {
		// A request without user content cannot be narrowed safely. Preserve the
		// established full-snapshot behavior for unusual protocol payloads.
		return latestUserFirstSegments(normalized)
	}
	latestUserEnd := latestUserStart
	for latestUserEnd < len(normalized) && isUserSegment(normalized[latestUserEnd]) {
		latestUserEnd++
	}
	currentUserText := make([]string, 0, latestUserEnd-latestUserStart)
	for _, segment := range normalized[latestUserStart:latestUserEnd] {
		currentUserText = append(currentUserText, segment.text)
	}
	// A single client turn may have several text content parts. Keep it in one
	// priority segment so every part of the latest input is scanned before the
	// prior output begins.
	selected := []promptSegment{{text: strings.Join(currentUserText, "\n\n"), user: true, role: "user"}}
	for index := latestUserStart - 1; index >= 0; index-- {
		if !isAssistantOutputSegment(normalized[index]) {
			continue
		}
		start := index
		for start > 0 && isAssistantOutputSegment(normalized[start-1]) {
			start--
		}
		selected = append(selected, normalized[start:index+1]...)
		break
	}
	return selected
}

// latestUserFirstSegments 与 normalizeSegmentsLatestUserFirst 的文本排序一致
// （最新 user 段在前；无 user 段时最后一段在前），但保留角色信息，供 blocking
// 收窄的退化分支在角色过滤前保持与异步路径相同的段顺序。
func latestUserFirstSegments(values []promptSegment) []promptSegment {
	normalized := normalizedPromptSegments(values)
	if len(normalized) == 0 {
		return nil
	}
	priorityIndex := len(normalized) - 1
	for index := len(normalized) - 1; index >= 0; index-- {
		if isUserSegment(normalized[index]) {
			priorityIndex = index
			break
		}
	}
	result := make([]promptSegment, 0, len(normalized))
	result = append(result, normalized[priorityIndex])
	for index, segment := range normalized {
		if index != priorityIndex {
			result = append(result, segment)
		}
	}
	return result
}

// filterSegmentsByAuditRoles 按审计角色过滤段，保持相对顺序；角色大小写/空白
// 已由 effectiveAuditRoles 归一化，此处只需精确匹配。
func filterSegmentsByAuditRoles(segments []promptSegment, auditRoles []string) []promptSegment {
	wanted := make(map[string]struct{}, len(auditRoles))
	for _, role := range auditRoles {
		wanted[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
	}
	result := make([]promptSegment, 0, len(segments))
	for _, segment := range segments {
		if _, ok := wanted[segment.role]; ok {
			result = append(result, segment)
		}
	}
	return result
}

func normalizedPromptSegments(values []promptSegment) []promptSegment {
	normalized := make([]promptSegment, 0, len(values))
	for _, value := range values {
		value.text = strings.TrimSpace(value.text)
		if value.text != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func latestUserSegmentStart(values []promptSegment) int {
	latest := -1
	for index := len(values) - 1; index >= 0; index-- {
		if isUserSegment(values[index]) {
			latest = index
			break
		}
	}
	for latest > 0 && isUserSegment(values[latest-1]) {
		latest--
	}
	return latest
}

func isUserSegment(segment promptSegment) bool {
	return segment.user || segment.role == "user"
}

func isAssistantOutputSegment(segment promptSegment) bool {
	return segment.role == "assistant" || segment.role == "model"
}

func promptSegmentTexts(values []promptSegment) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.text)
	}
	return result
}

func buildPrioritizedScanText(segments []string) (scanText string, metadataText string) {
	metadataText = strings.Join(segments, "\n\n")
	if len(segments) <= 1 {
		return metadataText, metadataText
	}
	return segments[0] + promptAuditPrioritySeparator + strings.Join(segments[1:], "\n\n"), metadataText
}

func promptSegmentsForRole(texts []string, role string) []promptSegment {
	result := make([]promptSegment, 0, len(texts))
	for _, text := range texts {
		result = append(result, promptSegment{text: text, user: role == "" || role == "user", role: role})
	}
	return result
}

func userPromptSegments(texts []string) []promptSegment {
	return promptSegmentsForRole(texts, "user")
}

func systemPromptSegments(texts []string) []promptSegment {
	return promptSegmentsForRole(texts, "system")
}

func RedactPreview(value string, maxRunes int) string {
	value = bearerPattern.ReplaceAllString(value, "Bearer ***")
	value = apiKeyPattern.ReplaceAllStringFunc(value, func(match string) string {
		if index := strings.IndexAny(match, ":= \t"); index >= 0 {
			return match[:index+1] + "***"
		}
		return "***"
	})
	value = canaryPattern.ReplaceAllString(value, "${1}***")
	value = emailPattern.ReplaceAllString(value, "***@***")
	value = phonePattern.ReplaceAllString(value, "***PHONE***")
	return TrimRunes(value, maxRunes)
}

// BuildPromptPreview stores only a short, non-recoverable head of sanitized
// input. Ordinary confidential prompts must not land nearly intact in PostgreSQL
// or the admin UI merely because no secret regex matched.
func BuildPromptPreview(value string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = DefaultPromptPreviewMaxRunes
	}
	redacted := strings.TrimSpace(RedactPreview(value, maxRunes))
	if redacted == "" {
		return ""
	}
	runes := []rune(redacted)
	hadTruncation := strings.HasSuffix(redacted, "…")
	if hadTruncation && len(runes) > 0 {
		runes = runes[:len(runes)-1]
	}
	if len(runes) == 0 {
		return "***…"
	}
	// Short unlabelled secrets would otherwise leak a recoverable prefix (e.g.
	// 20 runes → 5 visible). Fully withhold anything below the keep threshold.
	const minLengthForPartialPreview = 32
	if len(runes) < minLengthForPartialPreview {
		if hadTruncation {
			return "***…"
		}
		return "***"
	}
	// Keep at most a quarter of the already-truncated text, and never more than
	// 24 runes, so the majority of prompt content is withheld by default.
	keep := len(runes) / 4
	if keep > 24 {
		keep = 24
	}
	preview := string(runes[:keep]) + "***"
	if hadTruncation || keep < len(runes) {
		preview += "…"
	}
	return preview
}

// BuildFullPrompt returns the complete prompt text for audit-event storage and
// admin review, without redaction. NUL bytes are stripped because PostgreSQL
// TEXT rejects them, and the result is capped at maxRunes.
func BuildFullPrompt(value string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = DefaultFullPromptMaxRunes
	}
	value = strings.ReplaceAll(value, "\x00", "")
	return TrimRunes(strings.TrimSpace(value), maxRunes)
}

// FullPromptFromScanText reconstructs the display prompt from the worker scan
// payload. buildPrioritizedScanText inserts exactly one priority separator
// between the prioritized segment and the remainder, so replacing it with the
// metadata joiner yields the original multi-segment text.
func FullPromptFromScanText(scanText string) string {
	return BuildFullPrompt(strings.ReplaceAll(scanText, promptAuditPrioritySeparator, "\n\n"), DefaultFullPromptMaxRunes)
}

func TrimRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
