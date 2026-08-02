package securityaudit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseQwen3GuardStrictAndPolicy(t *testing.T) {
	tests := []struct {
		name, output string
		enabled      []string
		decision     EventDecision
		action       Action
		wantErr      bool
	}{
		{"safe", "Safety: Safe\nCategories: None", AllScannerIDs, EventPass, ActionAllow, false},
		{"controversial", "Safety: Controversial\nCategories: Violent", AllScannerIDs, EventFlag, ActionWarn, false},
		{"controversial pii escalates", "Safety: Controversial\nCategories: PII", AllScannerIDs, EventCritical, ActionBlock, false},
		{"unsafe", "Safety: Unsafe\nCategories: Jailbreak", AllScannerIDs, EventCritical, ActionBlock, false},
		{"unknown unsafe", "Safety: Unsafe\nCategories: Future Risk", AllScannerIDs, EventCritical, ActionBlock, false},
		{"disabled unsafe warns", "Safety: Unsafe\nCategories: Violent", []string{"PII"}, EventFlag, ActionWarn, false},
		{"extra explanation", "Safety: Safe\nCategories: None\nThis is safe", AllScannerIDs, EventPass, ActionAllow, false},
		{"duplicate", "Safety: Safe\nSafety: Safe", AllScannerIDs, "", "", true},
		{"duplicate categories", "Safety: Safe\nCategories: None\nCategories: PII", AllScannerIDs, "", "", true},
		{"missing categories", "Safety: Safe\n", AllScannerIDs, "", "", true},
		{"unknown safety", "Safety: Maybe\nCategories: PII", AllScannerIDs, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQwen3Guard(tt.output, tt.enabled)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.decision, result.Decision)
			require.Equal(t, tt.action, result.Action)
		})
	}
}

func TestParseQwen3GuardIgnoresAuxiliaryResponseFields(t *testing.T) {
	result, err := ParseQwen3Guard("Safety: Unsafe\nCategories: Jailbreak\nRefusal: No", AllScannerIDs)
	require.NoError(t, err)
	require.Equal(t, "Unsafe", result.Safety)
	require.Equal(t, []string{"jailbreak"}, result.Categories)

	serialized, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(serialized), "Refusal")
	require.NotContains(t, string(serialized), "No")
}

func TestQwen3GuardOfficialCategoriesAliasesAndUnknownAreStable(t *testing.T) {
	official := "Violent, Non-violent Illegal Acts, Sexual Content or Sexual Acts, PII, Suicide & Self-Harm, Unethical Acts, Politically Sensitive Topics, Copyright Violation, Jailbreak"
	result, err := ParseQwen3Guard("Safety: Unsafe\nCategories: "+official, AllScannerIDs)
	require.NoError(t, err)
	require.Equal(t, AllScannerIDs, result.MatchedScanners)
	require.Empty(t, result.UnknownCategories)
	require.Equal(t, "priority", result.PolicyID)
	require.Equal(t, 1, result.PolicyVersion)

	aliases := map[string]string{
		"violence": "violent", "non_violent_illegal_acts": "non_violent_illegal_acts",
		"sexual": "sexual_content_or_sexual_acts", "personal identifiable information": "pii",
		"suicide/self harm": "suicide_and_self_harm", "unethical": "unethical_acts",
		"political": "politically_sensitive_topics", "copyright": "copyright_violation",
		"prompt injection": "jailbreak",
	}
	for alias, canonical := range aliases {
		require.Equal(t, canonical, NormalizeCategory(alias), alias)
	}

	const canary = "PROMPT_CANARY_RAW_UNKNOWN_CATEGORY"
	unknown, err := ParseQwen3Guard("Safety: Unsafe\nCategories: "+canary, AllScannerIDs)
	require.NoError(t, err)
	require.Len(t, unknown.UnknownCategories, 1)
	require.NotContains(t, unknown.UnknownCategories[0], "canary")
	require.NotContains(t, unknown.UnknownCategories[0], "raw")
	require.Contains(t, unknown.UnknownCategories[0], "unknown:")
}

func TestExtractOpenAIContentSupportsStringAndTextBlocks(t *testing.T) {
	content, err := extractOpenAIContent([]byte(`{"choices":[{"message":{"content":"Safety: Safe\nCategories: None"}}]}`))
	require.NoError(t, err)
	require.Equal(t, "Safety: Safe\nCategories: None", content)
	content, err = extractOpenAIContent([]byte(`{"choices":[{"message":{"content":[{"type":"text","text":"Safety: Safe"},{"type":"text","text":"Categories: None"}]}}]}`))
	require.NoError(t, err)
	require.Equal(t, "Safety: Safe\nCategories: None", content)
	for _, body := range []string{`{}`, `{"choices":[]}`, `{"choices":[{"message":{"content":null}}]}`} {
		_, err := extractOpenAIContent([]byte(body))
		require.Error(t, err)
	}
}

func TestAggregateRequiresEveryResult(t *testing.T) {
	_, err := AggregateResults([]*NormalizedResult{{Decision: EventPass, Action: ActionAllow}, nil}, 0)
	require.Error(t, err)
	result, err := AggregateResults([]*NormalizedResult{
		{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, Categories: []string{"pii"}},
		{Decision: EventCritical, RiskLevel: RiskCritical, Action: ActionBlock, Categories: []string{"jailbreak"}},
	}, 0)
	require.NoError(t, err)
	require.Equal(t, EventCritical, result.Decision)
	require.Equal(t, ActionBlock, result.Action)
	require.Equal(t, []string{"pii", "jailbreak"}, result.Categories)
}

func TestAggregateDeduplicatesFactsAndUsesMostSevereEndpointMetadata(t *testing.T) {
	result, err := AggregateResults([]*NormalizedResult{
		{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, Safety: "Safe", Categories: []string{"pii"}, MatchedScanners: []string{"pii"}, ScannerScores: map[string]float64{"pii": 0}, ScannerEvidence: map[string]string{"pii": "first"}, GuardEndpointID: "safe-node", ScannerVersion: "safe-version", PolicyID: "priority", PolicyVersion: 1},
		{Decision: EventCritical, RiskLevel: RiskCritical, Action: ActionBlock, Safety: "Unsafe", Categories: []string{"pii", "jailbreak"}, MatchedScanners: []string{"pii", "jailbreak"}, ScannerScores: map[string]float64{"pii": 1, "jailbreak": 1}, ScannerEvidence: map[string]string{"pii": "second", "jailbreak": "blocked"}, GuardEndpointID: "block-node", ScannerVersion: "block-version", PolicyID: "priority", PolicyVersion: 2},
	}, 7*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, []string{"pii", "jailbreak"}, result.Categories)
	require.Equal(t, []string{"pii", "jailbreak"}, result.MatchedScanners)
	require.Equal(t, "first", result.ScannerEvidence["pii"], "evidence is deterministically first-seen")
	require.Equal(t, "block-node", result.GuardEndpointID)
	require.Equal(t, "block-version", result.ScannerVersion)
	require.Equal(t, 2, result.PolicyVersion)
	require.Equal(t, 7, result.LatencyMS)
}

func TestIssueSummariesAreDeterministicRedactedDerivedDTOs(t *testing.T) {
	const canary = "PROMPT_CANARY_EVIDENCE_SECRET"
	result := NormalizedResult{
		Decision: EventCritical, RiskLevel: RiskCritical, Action: ActionBlock,
		Categories: []string{"jailbreak", "pii"}, MatchedScanners: []string{"pii"},
		ScannerScores: map[string]float64{"pii": 1}, ScannerEvidence: map[string]string{"pii": canary},
		UnknownCategories: []string{unknownCategoryID("future risk")},
	}
	summaries := BuildIssueSummaries(result)
	require.Len(t, summaries, 3, "known categories are not hidden merely because policy disabled one")
	raw, err := json.Marshal(summaries)
	require.NoError(t, err)
	require.NotContains(t, string(raw), canary)
	for _, summary := range summaries {
		require.NotEmpty(t, summary.Title)
		require.NotEmpty(t, summary.Description)
		require.NotEmpty(t, summary.Code)
		require.NotEmpty(t, summary.EvidenceHash)
	}
}

func TestParseQwen3GuardToleratesGeneralModelVerbosity(t *testing.T) {
	tests := []struct {
		name, output string
		decision     EventDecision
		action       Action
	}{
		{"leading blank lines", "\n\nSafety: Safe\n\nCategories: None\n", EventPass, ActionAllow},
		{"interleaved explanation", "好的，以下是审核结果：\nSafety: Unsafe\n这段文字包含越狱意图。\nCategories: Jailbreak\n希望对你有帮助。", EventCritical, ActionBlock},
		{"markdown bold key", "**Safety:** Unsafe\n**Categories:** PII", EventCritical, ActionBlock},
		{"markdown bold value", "Safety: **Unsafe**\nCategories: **Jailbreak**", EventCritical, ActionBlock},
		{"list markers", "- Safety: Safe\n- Categories: None", EventPass, ActionAllow},
		{"quote marker", "> Safety: Safe\n> Categories: None", EventPass, ActionAllow},
		{"trailing punctuation", "Safety: Safe。\nCategories: None。", EventPass, ActionAllow},
		{"trailing period on category", "Safety: Unsafe.\nCategories: Violent.", EventCritical, ActionBlock},
		{"uppercase key and value", "SAFETY: UNSAFE\nCATEGORIES: Violent", EventCritical, ActionBlock},
		{"refusal auxiliary line", "Safety: Safe\nCategories: None\nRefusal: No", EventPass, ActionAllow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQwen3Guard(tt.output, AllScannerIDs)
			require.NoError(t, err)
			require.Equal(t, tt.decision, result.Decision)
			require.Equal(t, tt.action, result.Action)
		})
	}

	// Safety 值非法仍然判 invalid
	_, err := ParseQwen3Guard("Safety: Maybe.\nCategories: None", AllScannerIDs)
	require.Error(t, err)
	// 重复行仍判 invalid
	_, err = ParseQwen3Guard("- Safety: Safe\n* Safety: Unsafe\nCategories: None", AllScannerIDs)
	require.Error(t, err)
}

func TestScanBuildsMessagesWithOptionalSystemPrompt(t *testing.T) {
	type capturedRequest struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	newGuardServer := func(t *testing.T, captured *capturedRequest) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(body, captured))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Safety: Safe\nCategories: None"}}]}`))
		}))
	}

	t.Run("without system prompt keeps single user message", func(t *testing.T) {
		var captured capturedRequest
		server := newGuardServer(t, &captured)
		defer server.Close()
		scanner := NewOpenAICompatibleScanner()
		result, err := scanner.Scan(context.Background(), ActiveEndpoint{
			ID: "ep-raw", BaseURL: server.URL, Model: "qwen3guard", TimeoutMS: 2000, Enabled: true,
		}, "hello", AllScannerIDs)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, captured.Messages, 1)
		require.Equal(t, "user", captured.Messages[0].Role)
		require.Equal(t, "hello", captured.Messages[0].Content)
	})

	t.Run("with system prompt prepends system message", func(t *testing.T) {
		var captured capturedRequest
		server := newGuardServer(t, &captured)
		defer server.Close()
		scanner := NewOpenAICompatibleScanner()
		result, err := scanner.Scan(context.Background(), ActiveEndpoint{
			ID: "ep-wrapped", BaseURL: server.URL, Model: "gpt-test", TimeoutMS: 2000,
			SystemPrompt: DefaultGuardSystemPrompt, Enabled: true,
		}, "hello", AllScannerIDs)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, captured.Messages, 2)
		require.Equal(t, "system", captured.Messages[0].Role)
		require.Equal(t, DefaultGuardSystemPrompt, captured.Messages[0].Content)
		require.Equal(t, "user", captured.Messages[1].Role)
		require.Equal(t, "hello", captured.Messages[1].Content)
	})

	t.Run("whitespace-only system prompt treated as empty", func(t *testing.T) {
		var captured capturedRequest
		server := newGuardServer(t, &captured)
		defer server.Close()
		scanner := NewOpenAICompatibleScanner()
		_, err := scanner.Scan(context.Background(), ActiveEndpoint{
			ID: "ep-blank", BaseURL: server.URL, Model: "qwen3guard", TimeoutMS: 2000,
			SystemPrompt: "   \n  ", Enabled: true,
		}, "hello", AllScannerIDs)
		require.NoError(t, err)
		require.Len(t, captured.Messages, 1)
	})
}
