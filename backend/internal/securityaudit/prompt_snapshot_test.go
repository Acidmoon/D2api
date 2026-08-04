package securityaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestExtractPromptSnapshotProtocols(t *testing.T) {
	tests := []struct {
		protocol, body, first string
		count                 int
	}{
		{"openai_chat_completions", `{"messages":[{"role":"user","content":"old"},{"role":"assistant","content":"assistant turn"},{"role":"user","content":[{"type":"text","text":"最新😀"}]}]}`, "最新😀", 1},
		{"openai_responses", `{"input":[{"role":"user","content":[{"type":"input_text","text":"response text"}]}]}`, "response text", 1},
		{"anthropic_messages", `{"messages":[{"role":"user","content":[{"type":"text","text":"claude"}]}]}`, "claude", 1},
		{"gemini", `{"contents":[{"role":"user","parts":[{"text":"gemini"},{"inline_data":{"data":"BASE64"}}]}]}`, "gemini", 1},
		{"openai_images", `{"prompt":"draw a cat","image":"BASE64SECRET"}`, "draw a cat", 1},
		{"responses_websocket", `{"type":"response.create","response":{"input":"turn two"}}`, "turn two", 1},
	}
	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			snapshot, err := ExtractPromptSnapshot(Request{Protocol: tt.protocol, Body: []byte(tt.body), Stage: "http"}, true, nil)
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(snapshot.ScanText, tt.first))
			require.Equal(t, tt.count, snapshot.MessageCount)
			require.Equal(t, utf8.RuneCountInString(metadataTextForTest(snapshot.ScanText)), snapshot.PromptLength)
			require.NotEmpty(t, snapshot.PromptHash)
			require.NotContains(t, snapshot.ScanText, "BASE64SECRET")
		})
	}
}

func TestSnapshotRedactsCanariesAndPreservesHashOfScanText(t *testing.T) {
	body := `{"messages":[{"role":"user","content":"PROMPT_CANARY_ABC123 email@example.com +86 138 0013 8000 Bearer AUTH_CANARY_XYZ sk-secretvalue123 password=supersecret123"}]}`
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: []byte(body)}, true, nil)
	require.NoError(t, err)
	require.NotContains(t, snapshot.RedactedPreview, "ABC123")
	require.NotContains(t, snapshot.RedactedPreview, "email@example.com")
	require.NotContains(t, snapshot.RedactedPreview, "AUTH_CANARY_XYZ")
	require.NotContains(t, snapshot.RedactedPreview, "secretvalue123")
	require.NotContains(t, snapshot.RedactedPreview, "supersecret123")
	require.NotContains(t, snapshot.RedactedPreview, "138 0013 8000")
	require.Contains(t, snapshot.ScanText, "PROMPT_CANARY_ABC123")
	require.NotEqual(t, snapshot.ScanText, snapshot.RedactedPreview)
	digest := sha256.Sum256([]byte(metadataTextForTest(snapshot.ScanText)))
	require.Equal(t, hex.EncodeToString(digest[:]), snapshot.PromptHash)
	require.Empty(t, snapshot.Redacted().ScanText)
}

func TestSnapshotFullPromptKeepsUnredactedText(t *testing.T) {
	body := `{"messages":[{"role":"user","content":"PROMPT_CANARY_ABC123 email@example.com sk-secretvalue123"}]}`
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: []byte(body)}, true, nil)
	require.NoError(t, err)
	// The full prompt is stored verbatim for admin review, unlike the preview.
	require.Contains(t, snapshot.FullPrompt, "PROMPT_CANARY_ABC123 email@example.com sk-secretvalue123")
	require.NotContains(t, snapshot.RedactedPreview, "PROMPT_CANARY_ABC123")
	require.Equal(t, snapshot.FullPrompt, snapshot.Redacted().FullPrompt)
}

func TestBuildFullPromptStripsNULAndTruncates(t *testing.T) {
	require.Equal(t, "abcd", BuildFullPrompt("ab\x00cd", 0))
	long := strings.Repeat("长", DefaultFullPromptMaxRunes+10)
	trimmed := BuildFullPrompt(long, DefaultFullPromptMaxRunes)
	require.Equal(t, DefaultFullPromptMaxRunes+1, utf8.RuneCountInString(trimmed))
	require.True(t, strings.HasSuffix(trimmed, "…"))
}

func TestFullPromptFromScanTextRestoresMultiSegmentLayout(t *testing.T) {
	scanText, metadataText := buildPrioritizedScanText([]string{"latest user", "system policy", "earlier user"})
	require.Contains(t, scanText, promptAuditPrioritySeparator)
	require.Equal(t, metadataText, FullPromptFromScanText(scanText))

	singleScan, singleMeta := buildPrioritizedScanText([]string{"only"})
	require.NotContains(t, singleScan, promptAuditPrioritySeparator)
	require.Equal(t, singleMeta, FullPromptFromScanText(singleScan))
}

func TestSplitRunesDoesNotSplitUTF8(t *testing.T) {
	chunks := SplitRunes("中文😀éabc", 2)
	require.Equal(t, []string{"中文", "😀e", "́a", "bc"}, chunks)
	for _, chunk := range chunks {
		require.True(t, utf8.ValidString(chunk))
	}
	require.Equal(t, "中文😀éabc", strings.Join(chunks, ""))
}

func TestSplitRunesKeepsPrioritySegmentIndependent(t *testing.T) {
	latest := "请帮我编写一篇黄色小说 名字你来取"
	history := strings.Repeat("AGENTS.md 项目约束。", 40)
	chunks := SplitRunes(latest+promptAuditPrioritySeparator+history, 128)
	require.Greater(t, len(chunks), 2)
	require.Equal(t, latest, chunks[0])
	require.Equal(t, history, strings.Join(chunks[1:], ""))
	for _, chunk := range chunks {
		require.NotContains(t, chunk, promptAuditPrioritySeparator)
	}
}

func TestPromptSnapshotLatestUserTextBlockIsOnePrioritizedSegment(t *testing.T) {
	body := []byte(`{
		"messages":[
			{"role":"user","content":"历史输入"},
			{"role":"assistant","content":"assistant client injection"},
			{"role":"tool","content":"tool client injection"},
			{"role":"user","content":[
				{"type":"text","text":"最新第一块😀"},
				{"type":"image_url","image_url":{"url":"data:image/png;base64,IMAGE_CANARY_BASE64"}},
				{"type":"text","text":"最新第二块é"}
			]}
		]
	}`)
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: body}, true, nil)
	require.NoError(t, err)
	// 默认只审 user：assistant/tool 客户端注入不再进入审计文本；AsyncLatestTurnOnly
	// 默认收窄到最新一条 user 消息（同一消息的多文本块全部保留）。
	require.Equal(t, 2, snapshot.MessageCount)
	require.True(t, strings.HasPrefix(snapshot.ScanText, "最新第二块é"+promptAuditPrioritySeparator))
	require.Contains(t, snapshot.ScanText, "最新第一块😀")
	require.NotContains(t, snapshot.ScanText, "历史输入")
	require.NotContains(t, snapshot.ScanText, "assistant client injection")
	require.NotContains(t, snapshot.ScanText, "tool client injection")
	require.NotContains(t, snapshot.ScanText, "IMAGE_CANARY_BASE64")
	require.Equal(t, utf8.RuneCountInString(metadataTextForTest(snapshot.ScanText)), snapshot.PromptLength)
}

func TestPromptSnapshotSeparatesAnthropicUserPromptFromHarnessBlocks(t *testing.T) {
	latest := "请帮我编写一篇黄色小说 名字你来取"
	agents := "# AGENTS.md instructions\n<INSTRUCTIONS>" + strings.Repeat("安全约束。", 80) + "</INSTRUCTIONS>"
	environment := "<environment_context><cwd>/workspace</cwd></environment_context>"
	body := []byte(`{"system":"system policy","messages":[{"role":"user","content":[` +
		`{"type":"text","text":` + string(mustJSON(t, agents)) + `},` +
		`{"type":"text","text":` + string(mustJSON(t, environment)) + `},` +
		`{"type":"text","text":` + string(mustJSON(t, latest)) + `}` +
		`]}]}`)

	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "anthropic_messages", Body: body}, true, nil)
	require.NoError(t, err)
	require.Equal(t, 3, snapshot.MessageCount)
	require.True(t, strings.HasPrefix(snapshot.ScanText, latest+promptAuditPrioritySeparator))
	require.True(t, strings.HasPrefix(snapshot.RedactedPreview, "请帮我编写一篇黄色小说"))

	chunks := SplitRunes(snapshot.ScanText, 128)
	require.Equal(t, latest, chunks[0])
	require.Contains(t, strings.Join(chunks[1:], ""), "# AGENTS.md instructions")
	require.Contains(t, strings.Join(chunks[1:], ""), "<environment_context>")
	require.NotContains(t, strings.Join(chunks, ""), promptAuditPrioritySeparator)
}

func TestPromptSnapshotResponsesShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "string", body: `{"input":"plain response input"}`, want: "plain response input"},
		{name: "message array", body: `{"input":[{"role":"assistant","content":"assistant turn"},{"role":"user","content":[{"type":"input_text","text":"message block"}]}]}`, want: "message block"},
		{name: "direct input text", body: `{"input":[{"type":"input_text","text":"direct block"}]}`, want: "direct block"},
		{name: "single object", body: `{"input":{"role":"user","content":[{"type":"input_text","text":"single object"}]}}`, want: "single object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_responses", Body: []byte(tt.body)}, true, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, metadataTextForTest(snapshot.ScanText))
		})
	}
}

func TestPromptSnapshotGeminiBatchShapesAndMediaExclusion(t *testing.T) {
	body := []byte(`{
		"contents":{"role":"user","parts":[{"text":"root content"},{"inlineData":{"data":"ROOT_BASE64"}}]},
		"instances":[{"prompt":"instance prompt"}],
		"requests":[
			{"contents":[{"role":"model","parts":[{"text":"ignore model"}]},{"role":"user","parts":[{"text":"nested user"}]}]},
			{"instances":[{"prompt":"nested instance"}]}
		]
	}`)
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "gemini", Body: body}, false, nil)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(snapshot.ScanText, "nested instance"))
	for _, expected := range []string{"root content", "instance prompt", "nested user", "nested instance"} {
		require.Contains(t, snapshot.ScanText, expected)
	}
	require.NotContains(t, snapshot.ScanText, "ROOT_BASE64")
	require.NotContains(t, snapshot.ScanText, "ignore model")
}

func TestPromptSnapshotMediaOnlyExtractsDeterministicTextPrompts(t *testing.T) {
	body := []byte(`{
		"prompt":"draw a lighthouse",
		"image":"data:image/png;base64,IMAGE_CANARY",
		"input":{"negative_prompt":"no fog","image_prompt":"https://example.test/input.png","prompt":"draw a lighthouse"},
		"request":{"lyrics":"ocean song","input":"` + strings.Repeat("A", 300) + `"},
		"images":[{"description":"nested textual direction","image_url":"https://example.test/image.png"}]
	}`)
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "grok_media", Body: body}, true, nil)
	require.NoError(t, err)
	require.Equal(t, 4, snapshot.MessageCount)
	for _, expected := range []string{"draw a lighthouse", "no fog", "ocean song", "nested textual direction"} {
		require.Contains(t, snapshot.ScanText, expected)
	}
	require.Equal(t, 1, strings.Count(snapshot.ScanText, "draw a lighthouse"))
	require.NotContains(t, snapshot.ScanText, "IMAGE_CANARY")
	require.NotContains(t, snapshot.ScanText, "example.test")
	require.NotContains(t, snapshot.ScanText, strings.Repeat("A", 100))
}

func TestResponsesWebSocketOnlyAuditsResponseCreateAndPreservesStage(t *testing.T) {
	for _, stage := range []string{"first_turn", "subsequent_turn"} {
		snapshot, err := ExtractPromptSnapshot(Request{
			Protocol: "openai_responses", Stage: stage,
			Body: []byte(`{"type":"response.create","response":{"model":"gpt-test","input":[{"role":"user","content":[{"type":"input_text","text":"ws turn"}]}]}}`),
		}, true, nil)
		require.NoError(t, err)
		require.Equal(t, "ws turn", snapshot.ScanText)
		require.Equal(t, stage, snapshot.Stage)
	}
	_, err := ExtractPromptSnapshot(Request{
		Protocol: "openai_responses", Stage: "subsequent_turn",
		Body: []byte(`{"type":"conversation.item.create","response":{"input":"must not scan this frame"}}`),
	}, true, nil)
	require.True(t, errors.Is(err, ErrNoPromptText))
}

func TestPromptSnapshotEmptyAndLongUnicodeInput(t *testing.T) {
	_, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"function","content":"not audited role"},{"role":"user","content":"  "}]}`)}, true, nil)
	require.True(t, errors.Is(err, ErrNoPromptText))

	latest := strings.Repeat("最新😀é", 80)
	history := strings.Repeat("历史中文", 80)
	body := []byte(`{"messages":[{"role":"user","content":` + string(mustJSON(t, history)) + `},{"role":"user","content":` + string(mustJSON(t, latest)) + `}]}`)
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: body}, true, nil)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(snapshot.ScanText, latest))
	chunks := SplitRunes(snapshot.ScanText, 127)
	require.Equal(t, strings.Replace(snapshot.ScanText, promptAuditPrioritySeparator, "", 1), strings.Join(chunks, ""))
	require.Equal(t, latest, chunks[0]+strings.Join(chunks[1:len(SplitRunes(latest, 127))], ""))
	for _, chunk := range chunks {
		require.LessOrEqual(t, len([]rune(chunk)), 127)
		require.True(t, utf8.ValidString(chunk))
	}
}

func TestPromptSnapshotIncludesConfiguredRoles(t *testing.T) {
	tests := []struct {
		name, protocol, body string
		want                 []string
	}{
		{
			name:     "openai system developer assistant tool",
			protocol: "openai_chat_completions",
			body:     `{"messages":[{"role":"system","content":"system jailbreak"},{"role":"developer","content":"developer policy"},{"role":"assistant","content":"assistant jailbreak"},{"role":"tool","content":"tool payload"},{"role":"user","content":"hello"}]}`,
			want:     []string{"system jailbreak", "developer policy", "assistant jailbreak", "tool payload", "hello"},
		},
		{
			name:     "openai system only",
			protocol: "openai_chat_completions",
			body:     `{"messages":[{"role":"system","content":"only system instruction"}]}`,
			want:     []string{"only system instruction"},
		},
		{
			name:     "responses instructions",
			protocol: "openai_responses",
			body:     `{"instructions":"response instructions","input":[{"role":"user","content":[{"type":"input_text","text":"user turn"}]}]}`,
			want:     []string{"response instructions", "user turn"},
		},
		{
			name:     "anthropic system",
			protocol: "anthropic_messages",
			body:     `{"system":"claude system","messages":[{"role":"user","content":[{"type":"text","text":"claude user"}]}]}`,
			want:     []string{"claude system", "claude user"},
		},
		{
			name:     "gemini systemInstruction",
			protocol: "gemini",
			body:     `{"systemInstruction":{"parts":[{"text":"gemini system"}]},"contents":[{"role":"user","parts":[{"text":"gemini user"}]}]}`,
			want:     []string{"gemini system", "gemini user"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, err := ExtractPromptSnapshot(Request{Protocol: tt.protocol, Body: []byte(tt.body)}, false, []string{"user", "system", "developer", "assistant", "tool"})
			require.NoError(t, err)
			for _, expected := range tt.want {
				require.Contains(t, snapshot.ScanText, expected)
			}
		})
	}
}

func TestPromptSnapshotDefaultAuditsUserOnly(t *testing.T) {
	tests := []struct {
		name, protocol, body string
		want, omitted        []string
	}{
		{
			name:     "chat skips system developer assistant tool",
			protocol: "openai_chat_completions",
			body:     `{"messages":[{"role":"system","content":"system jailbreak"},{"role":"developer","content":"developer policy"},{"role":"assistant","content":"assistant jailbreak"},{"role":"tool","content":"tool payload"},{"role":"user","content":"hello"}]}`,
			want:     []string{"hello"},
			omitted:  []string{"system jailbreak", "developer policy", "assistant jailbreak", "tool payload"},
		},
		{
			name:     "responses skips instructions",
			protocol: "openai_responses",
			body:     `{"instructions":"response instructions","input":[{"role":"user","content":[{"type":"input_text","text":"user turn"}]}]}`,
			want:     []string{"user turn"},
			omitted:  []string{"response instructions"},
		},
		{
			name:     "anthropic skips system",
			protocol: "anthropic_messages",
			body:     `{"system":"claude system","messages":[{"role":"user","content":[{"type":"text","text":"claude user"}]}]}`,
			want:     []string{"claude user"},
			omitted:  []string{"claude system"},
		},
		{
			name:     "gemini skips systemInstruction",
			protocol: "gemini",
			body:     `{"systemInstruction":{"parts":[{"text":"gemini system"}]},"contents":[{"role":"user","parts":[{"text":"gemini user"}]}]}`,
			want:     []string{"gemini user"},
			omitted:  []string{"gemini system"},
		},
		{
			name:     "system-only request has no user text",
			protocol: "openai_chat_completions",
			body:     `{"messages":[{"role":"system","content":"only system instruction"}]}`,
			want:     nil,
			omitted:  []string{"only system instruction"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, err := ExtractPromptSnapshot(Request{Protocol: tt.protocol, Body: []byte(tt.body)}, true, nil)
			if len(tt.want) == 0 {
				require.True(t, errors.Is(err, ErrNoPromptText))
				return
			}
			require.NoError(t, err)
			for _, expected := range tt.want {
				require.Contains(t, snapshot.ScanText, expected)
			}
			for _, omitted := range tt.omitted {
				require.NotContains(t, snapshot.ScanText, omitted)
			}
		})
	}
}

func TestPromptSnapshotKeepsAllUserTurnsWithLatestPrioritized(t *testing.T) {
	body := []byte(`{"messages":[
		{"role":"system","content":"system instruction"},
		{"role":"user","content":"first user turn"},
		{"role":"assistant","content":"assistant turn"},
		{"role":"user","content":"second user turn"},
		{"role":"tool","content":"tool turn"},
		{"role":"user","content":"latest user turn"}
	]}`)
	// 显式关闭 AsyncLatestTurnOnly：历史 user 全部保留，最新 user 仍为优先段。
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: body}, false, nil)
	require.NoError(t, err)
	require.Equal(t, 3, snapshot.MessageCount)
	require.True(t, strings.HasPrefix(snapshot.ScanText, "latest user turn"+promptAuditPrioritySeparator))
	require.Contains(t, snapshot.ScanText, "first user turn")
	require.Contains(t, snapshot.ScanText, "second user turn")
	require.NotContains(t, snapshot.ScanText, "system instruction")
	require.NotContains(t, snapshot.ScanText, "assistant turn")
	require.NotContains(t, snapshot.ScanText, "tool turn")
	require.Equal(t, utf8.RuneCountInString(metadataTextForTest(snapshot.ScanText)), snapshot.PromptLength)
}

func TestAsyncLatestTurnOnlyNarrowsToLatestUserTurnByDefault(t *testing.T) {
	tests := []struct {
		name, protocol, body, want string
		count                      int
		omitted                    []string
	}{
		{
			name:     "chat drops history keeps latest user",
			protocol: "openai_chat_completions",
			body:     `{"messages":[{"role":"system","content":"system policy"},{"role":"user","content":"old user turn"},{"role":"assistant","content":"assistant turn"},{"role":"user","content":"latest user turn"}]}`,
			want:     "latest user turn", count: 1,
			omitted: []string{"system policy", "old user turn", "assistant turn"},
		},
		{
			name:     "responses drops instructions and history",
			protocol: "openai_responses",
			body:     `{"instructions":"response instructions","input":[{"role":"assistant","content":"assistant turn"},{"role":"user","content":[{"type":"input_text","text":"latest response turn"}]}]}`,
			want:     "latest response turn", count: 1,
			omitted: []string{"response instructions", "assistant turn"},
		},
		{
			name:     "anthropic drops system and history",
			protocol: "anthropic_messages",
			body:     `{"system":"claude system","messages":[{"role":"user","content":[{"type":"text","text":"old claude turn"}]},{"role":"assistant","content":"assistant turn"},{"role":"user","content":[{"type":"text","text":"latest claude turn"}]}]}`,
			want:     "latest claude turn", count: 1,
			omitted: []string{"claude system", "old claude turn", "assistant turn"},
		},
		{
			name:     "gemini drops systemInstruction and history",
			protocol: "gemini",
			body:     `{"systemInstruction":{"parts":[{"text":"gemini system"}]},"contents":[{"role":"user","parts":[{"text":"old gemini turn"}]},{"role":"model","parts":[{"text":"model turn"}]},{"role":"user","parts":[{"text":"latest gemini turn"}]}]}`,
			want:     "latest gemini turn", count: 1,
			omitted: []string{"gemini system", "old gemini turn", "model turn"},
		},
		{
			name:     "gemini batch requests keeps last request user",
			protocol: "gemini",
			body:     `{"requests":[{"contents":[{"role":"user","parts":[{"text":"first request user"}]}]},{"contents":[{"role":"user","parts":[{"text":"second request user"}]}]}]}`,
			want:     "second request user", count: 1,
			omitted: []string{"first request user"},
		},
		{
			name:     "consecutive user messages keep only the last",
			protocol: "openai_chat_completions",
			body:     `{"messages":[{"role":"user","content":"first"},{"role":"user","content":"second"}]}`,
			want:     "second", count: 1,
			omitted: []string{"first"},
		},
		{
			name:     "multipart latest user message keeps every part",
			protocol: "openai_chat_completions",
			body:     `{"messages":[{"role":"user","content":"old turn"},{"role":"assistant","content":"assistant turn"},{"role":"user","content":[{"type":"text","text":"part one"},{"type":"text","text":"part two"}]}]}`,
			want:     "part two" + promptAuditPrioritySeparator + "part one", count: 2,
			omitted: []string{"old turn", "assistant turn"},
		},
		{
			name:     "trailing assistant still finds the latest user message",
			protocol: "openai_chat_completions",
			body:     `{"messages":[{"role":"user","content":"only user turn"},{"role":"assistant","content":"trailing assistant"}]}`,
			want:     "only user turn", count: 1,
			omitted: []string{"trailing assistant"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot, err := ExtractPromptSnapshot(Request{Protocol: tt.protocol, Body: []byte(tt.body)}, true, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, snapshot.ScanText)
			require.Equal(t, tt.count, snapshot.MessageCount)
			for _, omitted := range tt.omitted {
				require.NotContains(t, snapshot.ScanText, omitted)
			}
		})
	}
}

func TestAsyncLatestTurnOnlyRespectsAuditRoles(t *testing.T) {
	req := Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"user","content":"only user"}]}`)}
	// audit_roles 不含 user 时收窄结果为空 → ErrNoPromptText，与现有语义一致。
	_, err := ExtractPromptSnapshot(req, true, []string{"system"})
	require.True(t, errors.Is(err, ErrNoPromptText))

	// 收窄模式下即使配置了 system/assistant，也只保留最新 user 段。
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"system","content":"policy"},{"role":"user","content":"hello"}]}`)}, true, []string{"user", "system"})
	require.NoError(t, err)
	require.Equal(t, "hello", snapshot.ScanText)
	require.NotContains(t, snapshot.ScanText, "policy")
}

func TestBlockingPromptSnapshotLimitsInputToLatestUserAndPreviousOutput(t *testing.T) {
	tests := []struct {
		name, protocol, body, want string
		omitted                    []string
	}{
		{
			name:     "chat keeps multipart latest user and prior assistant",
			protocol: "openai_chat_completions",
			body: `{"messages":[
				{"role":"system","content":"system instruction"},
				{"role":"user","content":"older user input"},
				{"role":"assistant","content":"older assistant output"},
				{"role":"tool","content":"tool payload"},
				{"role":"assistant","content":"previous assistant output"},
				{"role":"user","content":[{"type":"text","text":"latest user first part"},{"type":"text","text":"latest user second part"}]}
			]}`,
			want:    "latest user first part\n\nlatest user second part" + promptAuditPrioritySeparator + "previous assistant output",
			omitted: []string{"system instruction", "older user input", "older assistant output", "tool payload"},
		},
		{
			name:     "gemini keeps prior model output",
			protocol: "gemini",
			body: `{"systemInstruction":{"parts":[{"text":"system instruction"}]},"contents":[
				{"role":"user","parts":[{"text":"older user input"}]},
				{"role":"model","parts":[{"text":"previous model output"}]},
				{"role":"user","parts":[{"text":"latest user input"}]}
			]}`,
			want:    "latest user input" + promptAuditPrioritySeparator + "previous model output",
			omitted: []string{"system instruction", "older user input"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := []string{"user", "system", "assistant", "tool"}
			if tt.protocol == "gemini" {
				// gemini 的 model 角色已加入 AuditRoleNames，此组合经配置路径可达。
				roles = []string{"user", "system", "model"}
			}
			snapshot, err := ExtractBlockingPromptSnapshot(Request{Protocol: tt.protocol, Body: []byte(tt.body)}, true, roles)
			require.NoError(t, err)
			require.Equal(t, tt.want, snapshot.ScanText)
			for _, omitted := range tt.omitted {
				require.NotContains(t, snapshot.ScanText, omitted)
			}
		})
	}
}

func TestBlockingPromptSnapshotDefaultRolesLatestTurnOnlyKeepsLatestUserOnly(t *testing.T) {
	req := Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[
		{"role":"system","content":"system instruction"},
		{"role":"user","content":"older user input"},
		{"role":"assistant","content":"previous assistant output"},
		{"role":"user","content":"latest user input"}
	]}`)}
	snapshot, err := ExtractBlockingPromptSnapshot(req, true, nil)
	require.NoError(t, err)
	// blocking 收窄先于角色过滤：默认 user-only 下，收窄出的前一轮 assistant
	// 输出被角色过滤丢弃，只保留最新 user 段；历史 user 不再被合并进同一段。
	require.Equal(t, "latest user input", snapshot.ScanText)
	require.Equal(t, 1, snapshot.MessageCount)
	require.NotContains(t, snapshot.ScanText, "system instruction")
	require.NotContains(t, snapshot.ScanText, "older user input")
	require.NotContains(t, snapshot.ScanText, "previous assistant output")

	// 显式配置 assistant 时同一输入恢复「最新 user + 前一轮输出」语义。
	explicit, err := ExtractBlockingPromptSnapshot(req, true, []string{"user", "assistant"})
	require.NoError(t, err)
	require.Equal(t, "latest user input"+promptAuditPrioritySeparator+"previous assistant output", explicit.ScanText)
}

func TestBlockingPromptSnapshotPreservesFullScopeByDefaultAndWithoutUserInput(t *testing.T) {
	req := Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"system","content":"system instruction"},{"role":"user","content":"older user input"},{"role":"assistant","content":"previous output"},{"role":"user","content":"latest user input"}]}`)}
	full, err := ExtractPromptSnapshot(req, false, []string{"user", "system", "assistant"})
	require.NoError(t, err)
	defaultBlocking, err := ExtractBlockingPromptSnapshot(req, false, []string{"user", "system", "assistant"})
	require.NoError(t, err)
	require.Equal(t, full, defaultBlocking)

	// 默认只审 user：纯 system/assistant 请求不再产生可审计文本。
	noUser := Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"system","content":"system instruction"},{"role":"assistant","content":"assistant output"}]}`)}
	_, err = ExtractPromptSnapshot(noUser, true, nil)
	require.True(t, errors.Is(err, ErrNoPromptText))
	_, err = ExtractBlockingPromptSnapshot(noUser, true, nil)
	require.True(t, errors.Is(err, ErrNoPromptText))

	// 显式配置 system/assistant 时恢复旧语义：无 user 输入退化为全量快照。
	fullWithoutUser, err := ExtractPromptSnapshot(noUser, false, []string{"user", "system", "assistant"})
	require.NoError(t, err)
	narrowWithoutUser, err := ExtractBlockingPromptSnapshot(noUser, true, []string{"user", "system", "assistant"})
	require.NoError(t, err)
	require.Equal(t, fullWithoutUser, narrowWithoutUser)
}

func TestPromptSnapshotTrimsHugeSingleUserMessageToCap(t *testing.T) {
	// 巨型单条 user 消息（20000 runes）在 async 收窄路径上被截断到
	// MaxPromptAuditRunes：MessageCount 不变（单段截断），截断文本前 8000
	// runes 与原文逐字符一致，尾部标记被丢弃。
	text := strings.Repeat("最新😀é", 4000) // 5 runes × 4000 = 20000 runes
	tail := "END_MARKER_SHOULD_BE_DROPPED"
	body := []byte(`{"messages":[{"role":"user","content":` + string(mustJSON(t, text+tail)) + `}]}`)
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: body}, true, nil)
	require.NoError(t, err)
	require.Equal(t, 1, snapshot.MessageCount)
	require.Equal(t, MaxPromptAuditRunes, utf8.RuneCountInString(snapshot.ScanText))
	require.Equal(t, MaxPromptAuditRunes, snapshot.PromptLength)
	require.Equal(t, string([]rune(text)[:MaxPromptAuditRunes]), snapshot.ScanText)
	require.NotContains(t, snapshot.ScanText, tail)
}

func TestPromptSnapshotTrimsCumulativeRunesAcrossSegments(t *testing.T) {
	// 多段累计超限（全量路径）：最新 user 段（优先段，4000）完整、次新 user
	// 段（3500）完整、最旧 user 段（5000）只保留开头 500 runes，累计恰为
	// MaxPromptAuditRunes；段间分隔符计入 PromptLength 但不计入累计上限。
	latest := strings.Repeat("甲", 4000)
	oldest := strings.Repeat("乙", 3500)
	middle := strings.Repeat("丙", 5000)
	body := []byte(`{"messages":[` +
		`{"role":"user","content":` + string(mustJSON(t, oldest)) + `},` +
		`{"role":"user","content":` + string(mustJSON(t, middle)) + `},` +
		`{"role":"user","content":` + string(mustJSON(t, latest)) + `}` +
		`]}`)
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: body}, false, nil)
	require.NoError(t, err)
	require.Equal(t, 3, snapshot.MessageCount)
	require.Equal(t,
		latest+promptAuditPrioritySeparator+oldest+"\n\n"+string([]rune(middle)[:500]),
		snapshot.ScanText)
	require.Equal(t, MaxPromptAuditRunes, utf8.RuneCountInString(latest)+utf8.RuneCountInString(oldest)+500)
	require.Equal(t, utf8.RuneCountInString(metadataTextForTest(snapshot.ScanText)), snapshot.PromptLength)
	require.Equal(t, MaxPromptAuditRunes+2*2, snapshot.PromptLength)
}

func TestPromptSnapshotKeepsSmallMessagesUntouched(t *testing.T) {
	// 正常小消息（累计 < MaxPromptAuditRunes）完全保留：不触发截断，
	// 段顺序与 MessageCount 均不变。
	body := []byte(`{"messages":[` +
		`{"role":"user","content":"first"},` +
		`{"role":"user","content":"second"},` +
		`{"role":"user","content":"third"}` +
		`]}`)
	snapshot, err := ExtractPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: body}, false, nil)
	require.NoError(t, err)
	require.Equal(t, 3, snapshot.MessageCount)
	require.Equal(t, "third"+promptAuditPrioritySeparator+"first\n\nsecond", snapshot.ScanText)
	require.Equal(t, utf8.RuneCountInString(metadataTextForTest(snapshot.ScanText)), snapshot.PromptLength)
}

func TestBlockingPromptSnapshotTrimsHugeInputToCap(t *testing.T) {
	// blocking（latestTurnOnly）路径同样受 MaxPromptAuditRunes 约束：巨型最新
	// user 消息截断到 8000 runes，前一轮 assistant 输出在默认 user-only 角色
	// 过滤下不进入审计文本。
	latest := strings.Repeat("最新😀é", 4000) // 20000 runes
	body := []byte(`{"messages":[` +
		`{"role":"user","content":` + string(mustJSON(t, latest+"BLOCKING_TAIL")) + `},` +
		`{"role":"assistant","content":"prior assistant output"}` +
		`]}`)
	snapshot, err := ExtractBlockingPromptSnapshot(Request{Protocol: "openai_chat_completions", Body: body}, true, nil)
	require.NoError(t, err)
	require.Equal(t, 1, snapshot.MessageCount)
	require.Equal(t, MaxPromptAuditRunes, snapshot.PromptLength)
	require.Equal(t, MaxPromptAuditRunes, utf8.RuneCountInString(snapshot.ScanText))
	require.Equal(t, string([]rune(latest)[:MaxPromptAuditRunes]), snapshot.ScanText)
	require.NotContains(t, snapshot.ScanText, "BLOCKING_TAIL")
	require.NotContains(t, snapshot.ScanText, "prior assistant output")
}

func TestBuildPromptPreviewWithholdsMajorityOfOrdinaryText(t *testing.T) {
	prompt := strings.Repeat("机密业务提示词内容", 40)
	preview := BuildPromptPreview(prompt, DefaultPromptPreviewMaxRunes)
	require.NotEmpty(t, preview)
	require.Contains(t, preview, "***")
	require.LessOrEqual(t, utf8.RuneCountInString(strings.TrimSuffix(strings.TrimSuffix(preview, "…"), "***")), 24)
	require.Less(t, utf8.RuneCountInString(preview), utf8.RuneCountInString(prompt)/2)
	require.NotContains(t, preview, prompt)
}

func TestBuildPromptPreviewFullyMasksShortUnlabelledSecrets(t *testing.T) {
	require.Equal(t, "***", BuildPromptPreview("short-secret-value!!", DefaultPromptPreviewMaxRunes))
	require.Equal(t, "***", BuildPromptPreview(strings.Repeat("a", 31), DefaultPromptPreviewMaxRunes))
	partial := BuildPromptPreview(strings.Repeat("b", 32), DefaultPromptPreviewMaxRunes)
	require.True(t, strings.HasPrefix(partial, "b"))
	require.Contains(t, partial, "***")
}

func mustJSON(t *testing.T, value string) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return raw
}

func metadataTextForTest(scanText string) string {
	return strings.Replace(scanText, promptAuditPrioritySeparator, "\n\n", 1)
}
