package securityaudit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// --- FindRecentDecisionByPromptHash（sqlmock：参数化查询 + 命中/未命中/错误） ---
// 查询按 user_id + prompt_hash + created_at >= since + config_version + decision 非空
// 过滤；sqlmock 断言 4 个参数（含 config_version）都被原样传入。

func TestFindRecentDecisionByPromptHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	repo := NewPostgreSQLRepository(db)
	ctx := context.Background()
	since := time.Now().UTC().Add(-10 * time.Minute)

	t.Run("hit returns latest decision", func(t *testing.T) {
		mock.ExpectQuery("SELECT decision FROM prompt_audit_events").
			WithArgs(int64(42), "abc123hash", since, int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"decision"}).AddRow("critical"))
		decision, found, err := repo.FindRecentDecisionByPromptHash(ctx, 42, "abc123hash", since, 7)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "critical", decision)
	})

	t.Run("miss returns found=false", func(t *testing.T) {
		mock.ExpectQuery("SELECT decision FROM prompt_audit_events").
			WithArgs(int64(42), "abc123hash", since, int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"decision"}))
		decision, found, err := repo.FindRecentDecisionByPromptHash(ctx, 42, "abc123hash", since, 7)
		require.NoError(t, err)
		require.False(t, found)
		require.Empty(t, decision)
	})

	t.Run("config_version mismatch does not hit", func(t *testing.T) {
		// 同一 user+hash 的结论在旧配置（version 5）下产生：以 version 5 查重命中；
		// 管理员换端点/加 scanner/改模型后 config_version 变为 7，同窗口内同内容
		// 必须不命中（SQL 层 config_version=$4 过滤掉旧版本行），继续正常审核。
		// sqlmock 校验第 4 个参数（config_version）确实传入查询。
		mock.ExpectQuery("SELECT decision FROM prompt_audit_events").
			WithArgs(int64(42), "abc123hash", since, int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"decision"}).AddRow("critical"))
		decision, found, err := repo.FindRecentDecisionByPromptHash(ctx, 42, "abc123hash", since, 5)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "critical", decision)

		mock.ExpectQuery("SELECT decision FROM prompt_audit_events").
			WithArgs(int64(42), "abc123hash", since, int64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"decision"}))
		decision, found, err = repo.FindRecentDecisionByPromptHash(ctx, 42, "abc123hash", since, 7)
		require.NoError(t, err)
		require.False(t, found)
		require.Empty(t, decision)
	})

	t.Run("database error propagates", func(t *testing.T) {
		mock.ExpectQuery("SELECT decision FROM prompt_audit_events").
			WithArgs(int64(42), "abc123hash", since, int64(7)).
			WillReturnError(errors.New("database down"))
		_, _, err := repo.FindRecentDecisionByPromptHash(ctx, 42, "abc123hash", since, 7)
		require.Error(t, err)
	})

	require.NoError(t, mock.ExpectationsWereMet())
	mock.ExpectClose()
}

// --- async Enqueue 查重 ---

func dedupAsyncConfig() ActiveConfig {
	cfg := asyncConfig()
	cfg.DedupEnabled = true
	cfg.DedupWindowMinutes = 10
	return cfg
}

func dedupAsyncRequest() Request {
	return Request{RequestID: "request-dedup", UserID: 42, Protocol: "openai_chat_completions",
		Body: []byte(`{"messages":[{"role":"user","content":"重复的最新提示词"}]}`)}
}

func TestEnqueuerDedupHitSkipsEnqueue(t *testing.T) {
	for _, decision := range []string{string(EventPass), string(EventCritical)} {
		t.Run("decision="+decision, func(t *testing.T) {
			trace := []string{}
			repo := &fakeJobRepository{trace: &trace, dedupDecision: decision, dedupFound: true}
			payload := &fakePayloadStore{trace: &trace, values: map[int64]string{}}
			metrics := NewAtomicMetrics()
			enqueuer := NewEnqueuer(&fakeConfigStore{cfg: dedupAsyncConfig(), active: true}, repo, payload, metrics)
			require.NoError(t, enqueuer.Enqueue(context.Background(), dedupAsyncRequest()))
			// 命中确定性结论：跳过 enqueue，不产生任何 staging/payload/publish 动作。
			require.Empty(t, trace)
			// async 查重命中计入独立命中计数（仪表盘可度量功能收益）。
			require.Equal(t, int64(1), metrics.Snapshot().DedupHits)
		})
	}
}

func TestEnqueuerDedupMissAndErrorFailOpenToNormalEnqueue(t *testing.T) {
	t.Run("miss proceeds", func(t *testing.T) {
		trace := []string{}
		repo := &fakeJobRepository{trace: &trace, dedupFound: false, createJob: &Job{ID: 51}}
		payload := &fakePayloadStore{trace: &trace, values: map[int64]string{}}
		enqueuer := NewEnqueuer(&fakeConfigStore{cfg: dedupAsyncConfig(), active: true}, repo, payload)
		require.NoError(t, enqueuer.Enqueue(context.Background(), dedupAsyncRequest()))
		require.Equal(t, []string{"create_staging", "payload_set", "publish_queued"}, trace)
	})

	t.Run("database error fails open", func(t *testing.T) {
		trace := []string{}
		repo := &fakeJobRepository{trace: &trace, dedupErr: errors.New("database down"), createJob: &Job{ID: 52}}
		payload := &fakePayloadStore{trace: &trace, values: map[int64]string{}}
		enqueuer := NewEnqueuer(&fakeConfigStore{cfg: dedupAsyncConfig(), active: true}, repo, payload)
		require.NoError(t, enqueuer.Enqueue(context.Background(), dedupAsyncRequest()))
		require.Equal(t, []string{"create_staging", "payload_set", "publish_queued"}, trace)
	})

	t.Run("flag decision is not reusable", func(t *testing.T) {
		trace := []string{}
		repo := &fakeJobRepository{trace: &trace, dedupDecision: string(EventFlag), dedupFound: true, createJob: &Job{ID: 53}}
		payload := &fakePayloadStore{trace: &trace, values: map[int64]string{}}
		enqueuer := NewEnqueuer(&fakeConfigStore{cfg: dedupAsyncConfig(), active: true}, repo, payload)
		require.NoError(t, enqueuer.Enqueue(context.Background(), dedupAsyncRequest()))
		require.Equal(t, []string{"create_staging", "payload_set", "publish_queued"}, trace)
	})
}

func TestEnqueuerDedupDisabledOrNoUserProceeds(t *testing.T) {
	t.Run("dedup disabled proceeds", func(t *testing.T) {
		trace := []string{}
		cfg := dedupAsyncConfig()
		cfg.DedupEnabled = false
		repo := &fakeJobRepository{trace: &trace, dedupDecision: string(EventPass), dedupFound: true, createJob: &Job{ID: 54}}
		payload := &fakePayloadStore{trace: &trace, values: map[int64]string{}}
		enqueuer := NewEnqueuer(&fakeConfigStore{cfg: cfg, active: true}, repo, payload)
		require.NoError(t, enqueuer.Enqueue(context.Background(), dedupAsyncRequest()))
		require.Equal(t, []string{"create_staging", "payload_set", "publish_queued"}, trace)
	})

	t.Run("no user context proceeds without lookup", func(t *testing.T) {
		trace := []string{}
		repo := &fakeJobRepository{trace: &trace, dedupDecision: string(EventPass), dedupFound: true, createJob: &Job{ID: 55}}
		payload := &fakePayloadStore{trace: &trace, values: map[int64]string{}}
		enqueuer := NewEnqueuer(&fakeConfigStore{cfg: dedupAsyncConfig(), active: true}, repo, payload)
		req := dedupAsyncRequest()
		req.UserID = 0
		require.NoError(t, enqueuer.Enqueue(context.Background(), req))
		require.Equal(t, []string{"create_staging", "payload_set", "publish_queued"}, trace)
	})
}

// --- blocking GuardEvaluator 查重 ---

func dedupGuardConfig() ActiveConfig {
	cfg := guardConfig(ActiveEndpoint{ID: "one", Enabled: true, TimeoutMS: 1000, InputLimit: 100})
	cfg.DedupEnabled = true
	cfg.DedupWindowMinutes = 10
	return cfg
}

func dedupGuardSnapshot() PromptSnapshot {
	return PromptSnapshot{RequestID: "r", UserID: 42, PromptHash: "hash-123", ScanText: "review me", RedactedPreview: "rev***", PromptLength: 9}
}

func TestGuardEvaluatorDedupShortCircuitsBeforeScan(t *testing.T) {
	for _, tt := range []struct {
		name     string
		stored   string
		expected DecisionKind
	}{
		{name: "pass reuses allow", stored: string(EventPass), expected: DecisionAllow},
		{name: "critical reuses block", stored: string(EventCritical), expected: DecisionBlock},
	} {
		t.Run(tt.name, func(t *testing.T) {
			scannerCalls := 0
			scanner := PromptScannerFunc(func(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
				scannerCalls++
				return &NormalizedResult{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{}}, nil
			})
			repo := &fakeJobRepository{dedupDecision: tt.stored, dedupFound: true}
			metrics := NewAtomicMetrics()
			evaluator := newGuardEvaluator(scanner, repo, metrics, 2, 2)
			decision, err := evaluator.Evaluate(context.Background(), dedupGuardConfig(), dedupGuardSnapshot())
			require.NoError(t, err)
			require.Equal(t, tt.expected, decision.Kind)
			require.Zero(t, scannerCalls, "查重命中后不得调用审核模型")
			require.Zero(t, repo.recordBlockingCalls, "查重命中不重复落事件")
			// 短路路径同样计入决策指标（kind=allow/block，latency≈0）与独立命中计数。
			snapshot := metrics.Snapshot()
			require.Equal(t, int64(1), snapshot.DedupHits, "查重命中计入独立命中计数")
			require.Equal(t, int64(1), snapshot.Total, "查重短路也计入决策总量")
			if tt.expected == DecisionAllow {
				require.Equal(t, int64(1), snapshot.Allowed)
				require.Zero(t, snapshot.Blocked)
			} else {
				require.Equal(t, int64(1), snapshot.Blocked)
				require.Zero(t, snapshot.Allowed)
			}
		})
	}
}

func TestGuardEvaluatorDedupMissFlagAndErrorFallThroughToScan(t *testing.T) {
	t.Run("miss scans normally", func(t *testing.T) {
		scannerCalls := 0
		scanner := PromptScannerFunc(func(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
			scannerCalls++
			return &NormalizedResult{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{}}, nil
		})
		repo := &fakeJobRepository{dedupFound: false}
		evaluator := newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2)
		_, err := evaluator.Evaluate(context.Background(), dedupGuardConfig(), dedupGuardSnapshot())
		require.NoError(t, err)
		require.Equal(t, 1, scannerCalls)
	})

	t.Run("flag decision is not reusable", func(t *testing.T) {
		scannerCalls := 0
		scanner := PromptScannerFunc(func(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
			scannerCalls++
			return &NormalizedResult{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{}}, nil
		})
		repo := &fakeJobRepository{dedupDecision: string(EventFlag), dedupFound: true}
		evaluator := newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2)
		_, err := evaluator.Evaluate(context.Background(), dedupGuardConfig(), dedupGuardSnapshot())
		require.NoError(t, err)
		require.Equal(t, 1, scannerCalls)
	})

	t.Run("database error fails open to scan", func(t *testing.T) {
		scannerCalls := 0
		scanner := PromptScannerFunc(func(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
			scannerCalls++
			return &NormalizedResult{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{}}, nil
		})
		repo := &fakeJobRepository{dedupErr: errors.New("database down")}
		evaluator := newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2)
		_, err := evaluator.Evaluate(context.Background(), dedupGuardConfig(), dedupGuardSnapshot())
		require.NoError(t, err)
		require.Equal(t, 1, scannerCalls)
	})

	t.Run("dedup disabled scans normally", func(t *testing.T) {
		scannerCalls := 0
		scanner := PromptScannerFunc(func(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
			scannerCalls++
			return &NormalizedResult{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{}}, nil
		})
		repo := &fakeJobRepository{dedupDecision: string(EventPass), dedupFound: true}
		cfg := dedupGuardConfig()
		cfg.DedupEnabled = false
		evaluator := newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2)
		_, err := evaluator.Evaluate(context.Background(), cfg, dedupGuardSnapshot())
		require.NoError(t, err)
		require.Equal(t, 1, scannerCalls)
	})

	t.Run("no user context scans normally", func(t *testing.T) {
		scannerCalls := 0
		scanner := PromptScannerFunc(func(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
			scannerCalls++
			return &NormalizedResult{Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{}}, nil
		})
		repo := &fakeJobRepository{dedupDecision: string(EventPass), dedupFound: true}
		snapshot := dedupGuardSnapshot()
		snapshot.UserID = 0
		evaluator := newGuardEvaluator(scanner, repo, NewAtomicMetrics(), 2, 2)
		_, err := evaluator.Evaluate(context.Background(), dedupGuardConfig(), snapshot)
		require.NoError(t, err)
		require.Equal(t, 1, scannerCalls)
	})
}

func TestReusableDedupDecision(t *testing.T) {
	require.True(t, ReusableDedupDecision(string(EventPass)))
	require.True(t, ReusableDedupDecision(string(EventCritical)))
	require.False(t, ReusableDedupDecision(string(EventFlag)))
	require.False(t, ReusableDedupDecision(""))
	require.False(t, ReusableDedupDecision("allow"))
}

func TestDedupShortCircuitDecision(t *testing.T) {
	allow := DedupShortCircuitDecision(string(EventPass))
	require.NotNil(t, allow)
	require.Equal(t, DecisionAllow, allow.Kind)
	require.True(t, allow.AllowNextStage)

	block := DedupShortCircuitDecision(string(EventCritical))
	require.NotNil(t, block)
	require.Equal(t, DecisionBlock, block.Kind)
	require.Equal(t, ErrorCodeBlocked, block.ErrorCode)
	require.False(t, block.AllowNextStage)

	require.Nil(t, DedupShortCircuitDecision(string(EventFlag)))
	require.Nil(t, DedupShortCircuitDecision(""))
}
