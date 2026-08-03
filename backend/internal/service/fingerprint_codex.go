package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/tidwall/gjson"
)

// Codex OAuth（ChatGPT 订阅）账号的探测构造。
// 这类账号不能走 api.openai.com：必须打 chatgpt.com 内部 Codex 端点并携带
// 专用身份头（originator 与 User-Agent 首段配套，否则上游 404，issue #3901）。
// 构造口径与账号测试连接（account_test_service.go 的 OAuth 分支）保持一致。

// fingerprintResponseMaxBytes 单次探测响应最大读取字节。
// Codex 流式响应含 reasoning 加密内容回显，比监控模块的 64KB 预算大。
const fingerprintResponseMaxBytes = 256 * 1024

// postFingerprintRawJSON 与 postRawJSON 相同（SSRF 安全客户端），仅响应体上限不同；
// 额外返回响应头（429 的 Retry-After 解析需要）。
func postFingerprintRawJSON(ctx context.Context, fullURL string, payload []byte, headers map[string]string) ([]byte, int, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := monitorHTTPClient.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, fingerprintResponseMaxBytes))
	if err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("read body: %w", err)
	}
	return respBody, resp.StatusCode, resp.Header, nil
}

// parseFingerprintRetryAfter 解析 429 响应的 Retry-After 头（秒数或 HTTP 日期），
// 等待时长按 fingerprintRetryAfterCap 截断；缺失/无法解析返回 false。
func parseFingerprintRetryAfter(h http.Header) (time.Duration, bool) {
	v := strings.TrimSpace(h.Get("Retry-After"))
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		return min(time.Duration(secs)*time.Second, fingerprintRetryAfterCap), true
	}
	if ts, err := http.ParseTime(v); err == nil {
		d := time.Until(ts)
		if d <= 0 {
			return 0, true
		}
		return min(d, fingerprintRetryAfterCap), true
	}
	return 0, false
}

// buildCodexFingerprintBody 构造 Codex Responses 探测请求体。
// 与账号测试连接一致：instructions 用 Codex base prompt（openai.DefaultInstructions），
// input 为 Responses 消息数组，stream=true、store=false。
// 一词回答约束并入用户消息文本（instructions 必须是 Codex base prompt，不能替换）。
//
// temperature 刻意不发送：Codex/gpt-5 系对非默认 temperature 敏感（可能 400），
// Codex CLI 真实流量也不带该字段；默认温度即 1.0，正好是指纹定义温度。
// T=0 探测因此实际也按默认温度采样，t0 快信号降级为「同温度答案一致性」对比（双侧一致，仍有效）。
//
// max_output_tokens 与 reasoning 都可能被上游拒绝（400，schema 窄）：被拒后由
// runProbeOnce 置位对应标记，后续请求省略该字段重试——与真实转发
// openai_gateway_forward 的 rejected-field retry 行为一致。
func buildCodexFingerprintBody(t *fingerprintProbeTarget, prompt string, _ float64) ([]byte, error) {
	body := map[string]any{
		"model":        t.model,
		"instructions": openai.DefaultInstructions,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": fingerprintSystemPrompt + " " + prompt},
				},
			},
		},
		"stream": true,
		"store":  false,
	}
	// 上游拒绝 max_output_tokens 字段后省略（省略，不判不适用）。
	if !t.maxOutputTokensUnsupported.Load() {
		body["max_output_tokens"] = fingerprintMaxTokens
	}
	// reasoning 尽量关；上游拒绝该字段时（400 且提到 reasoning/effort）由
	// codexReasoningUnsupported 标记后续省略（省略，不判不适用）。
	if !t.codexReasoningUnsupported.Load() {
		body["reasoning"] = map[string]any{"effort": "none"}
	}
	return json.Marshal(body)
}

// codexFingerprintHeaders 构造 Codex OAuth 请求头（参照 account_test_service 的 OAuth 分支）：
// Bearer access_token + chatgpt-account-id + Codex 身份头（originator 与 UA 首段配套）。
func codexFingerprintHeaders(t *fingerprintProbeTarget) map[string]string {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+t.apiKey)
	h.Set("accept", "text/event-stream") // stream=true 必须
	if t.chatgptAccountID != "" {
		h.Set("chatgpt-account-id", t.chatgptAccountID)
	}
	if t.customUserAgent != "" {
		h.Set("user-agent", t.customUserAgent)
	}
	ensureCodexIdentityHeaders(h)  // 补齐 UA / originator / version / OpenAI-Beta
	enforceCodexIdentityHeaders(h) // originator 与最终 UA 首段配套（防 404）
	out := make(map[string]string, len(h))
	for key := range h {
		out[key] = h.Get(key)
	}
	return out
}

// extractFingerprintCodexSSE 解析 Codex 流式响应：取 response.completed 事件的完整
// response 对象，复用 extractOpenAIResponsesText 抽文本、usage.output_tokens 取输出 token 数。
// 返回 (text, completionTokens, errMsg)；errMsg 非空表示本次流式失败。
func extractFingerprintCodexSSE(body []byte) (string, int, string) {
	var completed []byte
	streamErr := ""
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}
		payload := sseDataPrefix.ReplaceAllString(line, "")
		if payload == "[DONE]" {
			break
		}
		payloadBytes := []byte(payload)
		switch gjson.GetBytes(payloadBytes, "type").String() {
		case "response.completed", "response.done":
			completed = []byte(gjson.GetBytes(payloadBytes, "response").Raw)
		case "response.failed":
			streamErr = gjson.GetBytes(payloadBytes, "response.error.message").String()
			if streamErr == "" {
				streamErr = "codex response failed"
			}
		case "error":
			streamErr = gjson.GetBytes(payloadBytes, "error.message").String()
			if streamErr == "" {
				streamErr = "codex stream error"
			}
		}
	}
	if streamErr != "" {
		return "", 0, streamErr
	}
	if len(completed) == 0 {
		return "", 0, "stream ended before response.completed"
	}
	return extractOpenAIResponsesText(completed), int(gjson.GetBytes(completed, "usage.output_tokens").Int()), ""
}
