package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tidwall/gjson"
)

// healthyFingerprintResults 构造一轮全部达标的电池结果（16 cell 各 12 个有效样本）。
func healthyFingerprintResults() map[string]*fingerprintCellResult {
	results := make(map[string]*fingerprintCellResult)
	for _, c := range fingerprintCells() {
		res := newFingerprintCellResult()
		for i := 0; i < 12; i++ {
			res.valid++
			res.counts["42"]++
			res.samples = append(res.samples, "42")
		}
		results[c.Key()] = res
	}
	return results
}

// failedFingerprintResults 构造一轮全灭的电池结果（所有探测重试后失败）。
func failedFingerprintResults() map[string]*fingerprintCellResult {
	results := make(map[string]*fingerprintCellResult)
	for _, c := range fingerprintCells() {
		res := newFingerprintCellResult()
		res.failures = fingerprintSamplesPerCell + fingerprintGreedySamplesPerCell
		results[c.Key()] = res
	}
	return results
}

// 空采样注册失败：不写参考文件。
func TestFinishReferenceRegistration_EmptySamplingNoFile(t *testing.T) {
	svc := &FingerprintService{store: newFingerprintStore(t.TempDir())}
	accountID := int64(1643)

	err := svc.finishReferenceRegistration("gpt-5.4", &accountID, failedFingerprintResults(), "upstream HTTP 404: not found")
	if err == nil {
		t.Fatal("空采样注册应返回错误")
	}
	if !strings.Contains(err.Error(), "upstream HTTP 404") {
		t.Fatalf("错误应包含最后上游错误摘要，实际：%s", err.Error())
	}
	refs, listErr := svc.store.listReferences()
	if listErr != nil {
		t.Fatalf("listReferences: %v", listErr)
	}
	if len(refs) != 0 {
		t.Fatalf("空采样不应写出参考文件，实际 %d 个", len(refs))
	}
}

// 空采样注册失败：已有同名旧参考文件时保留旧的，不被空参考覆盖。
func TestFinishReferenceRegistration_EmptySamplingKeepsOldFile(t *testing.T) {
	svc := &FingerprintService{store: newFingerprintStore(t.TempDir())}
	accountID := int64(1643)

	// 先健康注册一次。
	if err := svc.finishReferenceRegistration("gpt-5.4", &accountID, healthyFingerprintResults(), ""); err != nil {
		t.Fatalf("健康注册应成功: %v", err)
	}
	// 再跑一轮全灭的：应失败且旧文件保持不变。
	if err := svc.finishReferenceRegistration("gpt-5.4", &accountID, failedFingerprintResults(), "upstream HTTP 401"); err == nil {
		t.Fatal("空采样注册应返回错误")
	}
	ref, err := svc.store.loadReference("gpt-5.4")
	if err != nil {
		t.Fatalf("loadReference: %v", err)
	}
	if fingerprintReferenceEmpty(ref) {
		t.Fatal("旧参考文件被空采样覆盖")
	}
	if got := ref.Cells["random_number_1_100|en"].Valid; got != 12 {
		t.Fatalf("旧参考内容应保留（valid=12），实际 %d", got)
	}
}

func TestCheckFingerprintReferenceQuality(t *testing.T) {
	// 8 个 cell 达标（≥10 有效样本）即通过。
	results := failedFingerprintResults()
	i := 0
	for _, c := range fingerprintCells() {
		if i >= fingerprintMinCells {
			break
		}
		res := results[c.Key()]
		res.valid = fingerprintMinValidSamples
		res.failures = 0
		i++
	}
	if err := checkFingerprintReferenceQuality(results, ""); err != nil {
		t.Fatalf("8 项达标应通过: %v", err)
	}

	// 全部失败 → 错误包含比例与最后上游错误。
	err := checkFingerprintReferenceQuality(failedFingerprintResults(), "upstream HTTP 500: boom")
	if err == nil {
		t.Fatal("全灭采样应返回错误")
	}
	if !strings.Contains(err.Error(), "0/16") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("错误信息不完整：%s", err.Error())
	}
}

func TestFingerprintReferenceEmpty(t *testing.T) {
	if !fingerprintReferenceEmpty(nil) {
		t.Fatal("nil 参考应视为空")
	}
	if !fingerprintReferenceEmpty(&FingerprintReference{Cells: map[string]*FingerprintReferenceCell{}}) {
		t.Fatal("无 cell 的参考应视为空")
	}
	zero := &FingerprintReference{Cells: map[string]*FingerprintReferenceCell{
		"random_number_1_100|en": {Samples: 15, Valid: 0, Distribution: map[string]int{}},
	}}
	if !fingerprintReferenceEmpty(zero) {
		t.Fatal("所有 cell valid=0 应视为空")
	}
	nonEmpty := &FingerprintReference{Cells: map[string]*FingerprintReferenceCell{
		"random_number_1_100|en": {Samples: 15, Valid: 3, Distribution: map[string]int{"42": 3}},
	}}
	if fingerprintReferenceEmpty(nonEmpty) {
		t.Fatal("有 valid 样本的参考不应视为空")
	}
}

// Codex 探测请求体：stream/store/instructions/input 结构与 reasoning 省略逻辑。
func TestBuildCodexFingerprintBody(t *testing.T) {
	target := &fingerprintProbeTarget{model: "gpt-5.4", codexOAuth: true}

	body, err := buildCodexFingerprintBody(target, "Name a random number between 1 and 100.", fingerprintProbeTemperature)
	if err != nil {
		t.Fatalf("buildCodexFingerprintBody: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("body 非合法 JSON: %v", err)
	}
	if m["model"] != "gpt-5.4" {
		t.Fatalf("model = %v", m["model"])
	}
	if m["stream"] != true || m["store"] != false {
		t.Fatalf("Codex 必须 stream=true, store=false，实际 stream=%v store=%v", m["stream"], m["store"])
	}
	if m["max_output_tokens"] != float64(fingerprintMaxTokens) {
		t.Fatalf("max_output_tokens = %v", m["max_output_tokens"])
	}
	if inst, _ := m["instructions"].(string); strings.TrimSpace(inst) == "" {
		t.Fatal("instructions 必须非空（Codex base prompt）")
	}
	if _, ok := m["temperature"]; ok {
		t.Fatal("Codex 路径不应发送 temperature 字段")
	}
	// input 为 Responses 消息数组，一词约束并入用户消息文本。
	input, ok := m["input"].([]any)
	if !ok || len(input) != 1 {
		t.Fatalf("input 结构错误: %v", m["input"])
	}
	msg, ok := input[0].(map[string]any)
	if !ok {
		t.Fatalf("input[0] 类型错误: %T", input[0])
	}
	content, ok := msg["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content 结构错误: %v", msg["content"])
	}
	part, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] 类型错误: %T", content[0])
	}
	text, _ := part["text"].(string)
	if !strings.Contains(text, fingerprintSystemPrompt) || !strings.Contains(text, "random number") {
		t.Fatalf("用户消息文本应含一词约束与探测 prompt，实际：%q", text)
	}
	// 默认带 reasoning effort none。
	if reasoning, ok := m["reasoning"].(map[string]any); !ok || reasoning["effort"] != "none" {
		t.Fatalf("默认应带 reasoning effort none: %v", m["reasoning"])
	}

	// 上游拒绝后省略 reasoning 字段。
	target.codexReasoningUnsupported.Store(true)
	body, err = buildCodexFingerprintBody(target, "Pick a color.", fingerprintProbeTemperature)
	if err != nil {
		t.Fatalf("buildCodexFingerprintBody: %v", err)
	}
	var m2 map[string]any
	if err := json.Unmarshal(body, &m2); err != nil {
		t.Fatalf("body 非合法 JSON: %v", err)
	}
	if _, ok := m2["reasoning"]; ok {
		t.Fatal("codexReasoningUnsupported 置位后应省略 reasoning 字段")
	}

	// 上游拒绝后省略 max_output_tokens 字段（与真实转发 rejected-field retry 一致）。
	target.codexReasoningUnsupported.Store(false)
	target.maxOutputTokensUnsupported.Store(true)
	body, err = buildCodexFingerprintBody(target, "Pick a color.", fingerprintProbeTemperature)
	if err != nil {
		t.Fatalf("buildCodexFingerprintBody: %v", err)
	}
	var m3 map[string]any
	if err := json.Unmarshal(body, &m3); err != nil {
		t.Fatalf("body 非合法 JSON: %v", err)
	}
	if _, ok := m3["max_output_tokens"]; ok {
		t.Fatal("maxOutputTokensUnsupported 置位后应省略 max_output_tokens 字段")
	}
}

// Codex 请求头：Bearer、chatgpt-account-id、Codex 身份头配套。
func TestCodexFingerprintHeaders(t *testing.T) {
	target := &fingerprintProbeTarget{
		codexOAuth:       true,
		apiKey:           "test-access-token",
		chatgptAccountID: "acc-123",
	}
	h := codexFingerprintHeaders(target)

	if got := h["Authorization"]; got != "Bearer test-access-token" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := h["Chatgpt-Account-Id"]; got != "acc-123" {
		t.Fatalf("chatgpt-account-id = %q", got)
	}
	if got := h["Originator"]; got != "codex_cli_rs" {
		t.Fatalf("originator = %q", got)
	}
	if got := h["User-Agent"]; got != codexCLIUserAgent {
		t.Fatalf("User-Agent = %q, want codex CLI UA", got)
	}
	if got := h["Openai-Beta"]; got != "responses=experimental" {
		t.Fatalf("OpenAI-Beta = %q", got)
	}
	if got := h["Accept"]; got != "text/event-stream" {
		t.Fatalf("accept = %q（stream=true 必须 SSE）", got)
	}
	if got := h["Version"]; got != codexCLIVersion {
		t.Fatalf("version = %q", got)
	}

	// 无 account-id 时不写该头。
	target.chatgptAccountID = ""
	h = codexFingerprintHeaders(target)
	if _, ok := h["Chatgpt-Account-Id"]; ok {
		t.Fatal("无 chatgpt-account-id 时不应写该头")
	}
}

// Codex SSE 解析：completed 事件取文本与 usage；failed / 缺 completed 报错。
func TestExtractFingerprintCodexSSE(t *testing.T) {
	body := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"42\"}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"42\"}]}],\"usage\":{\"output_tokens\":3}}}\n\n" +
		"data: [DONE]\n\n"
	text, tokens, serr := extractFingerprintCodexSSE([]byte(body))
	if serr != "" {
		t.Fatalf("正常流不应报错: %s", serr)
	}
	if text != "42" || tokens != 3 {
		t.Fatalf("text=%q tokens=%d, want 42/3", text, tokens)
	}

	failed := "data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"boom\"}}}\n\n"
	if _, _, serr := extractFingerprintCodexSSE([]byte(failed)); serr != "boom" {
		t.Fatalf("response.failed 应透传错误消息，实际 %q", serr)
	}

	noCompleted := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"4\"}\n\n"
	if _, _, serr := extractFingerprintCodexSSE([]byte(noCompleted)); serr == "" {
		t.Fatal("缺 response.completed 应报错")
	}
}

// 上游 400 拒绝 max_output_tokens 时：置位省略标记 + 本次可重试；
// 重试请求省略该字段后成功。覆盖 runProbeOnce 的 rejected-field 检测与
// buildBody Responses 分支的省略逻辑（与真实转发 openai_gateway_forward 一致）。
func TestRunProbeOnceResponsesMaxTokensRejected(t *testing.T) {
	// 临时替换 SSRF 安全 client 为普通 client，让 httptest (127.0.0.1) 可连通。
	orig := monitorHTTPClient
	monitorHTTPClient = &http.Client{Timeout: 5 * time.Second}
	t.Cleanup(func() { monitorHTTPClient = orig })

	var calls int
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, b)
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"Unsupported parameter: max_output_tokens"}`))
			return
		}
		_, _ = w.Write([]byte(`{"output":[{"type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"42"}]}]}`))
	}))
	defer srv.Close()

	target := &fingerprintProbeTarget{
		provider: MonitorProviderOpenAI,
		apiMode:  MonitorAPIModeResponses,
		baseURL:  srv.URL,
		apiKey:   "sk-test",
		model:    "gpt-5.4",
	}
	pacer := &fingerprintPacer{}
	ctx := context.Background()

	// 第一次：400 拒绝 max_output_tokens → 置位 + 可重试。
	outcome, perr := target.runProbeOnce(ctx, "Name a random number between 1 and 100.", fingerprintProbeTemperature, pacer)
	if perr == nil {
		t.Fatalf("首次探测应收到 400 错误，outcome=%+v", outcome)
	}
	if !perr.retryable {
		t.Fatalf("400 字段拒绝应标记可重试，实际 %v", perr)
	}
	if perr.statusCode != http.StatusBadRequest {
		t.Fatalf("statusCode = %d, want 400", perr.statusCode)
	}
	if !target.maxOutputTokensUnsupported.Load() {
		t.Fatal("400 后应置位 maxOutputTokensUnsupported")
	}
	if len(bodies) != 1 {
		t.Fatalf("body 数量 = %d, want 1", len(bodies))
	}
	if !gjson.ValidBytes(bodies[0]) {
		t.Fatalf("首个请求体非 JSON: %s", bodies[0])
	}
	if !gjson.GetBytes(bodies[0], "max_output_tokens").Exists() {
		t.Fatal("首个请求应携带 max_output_tokens")
	}

	// 重试：省略 max_output_tokens → 200 成功。
	outcome, perr = target.runProbeOnce(ctx, "Name a random number between 1 and 100.", fingerprintProbeTemperature, pacer)
	if perr != nil {
		t.Fatalf("重试不应失败: %v", perr)
	}
	if outcome.text != "42" {
		t.Fatalf("text = %q, want 42", outcome.text)
	}
	if len(bodies) != 2 {
		t.Fatalf("body 数量 = %d, want 2", len(bodies))
	}
	if gjson.GetBytes(bodies[1], "max_output_tokens").Exists() {
		t.Fatal("重试请求应省略 max_output_tokens")
	}
}
