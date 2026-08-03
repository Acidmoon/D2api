package accountguard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeConfigStore struct {
	cfg    securityaudit.ActiveConfig
	active bool
}

func (f *fakeConfigStore) Active() (securityaudit.ActiveConfig, bool) { return f.cfg, f.active }

type fakeEvaluator struct {
	decision *securityaudit.PromptDecision
	err      error
	calls    int
}

func (f *fakeEvaluator) Evaluate(_ context.Context, _ securityaudit.ActiveConfig, _ securityaudit.PromptSnapshot) (*securityaudit.PromptDecision, error) {
	f.calls++
	return f.decision, f.err
}

type banRecord struct {
	userID int64
	until  time.Time
	ttl    time.Duration
}

type fakeCounter struct {
	mu          sync.Mutex
	count       int64
	incremented int
	reset       int
	claimOK     bool
	incrErr     error
	bans        []banRecord
}

func (f *fakeCounter) IncrementViolationCount(_ context.Context, _ int64, _ time.Duration) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.incremented++
	if f.incrErr != nil {
		return 0, f.incrErr
	}
	f.count++
	return f.count, nil
}

func (f *fakeCounter) ResetViolationCount(_ context.Context, _ int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reset++
	f.count = 0
	return nil
}

func (f *fakeCounter) ClaimViolationNotifyCooldown(_ context.Context, _ int64, _ time.Duration) (bool, error) {
	return f.claimOK, nil
}

func (f *fakeCounter) SetUserViolationBan(_ context.Context, userID int64, until time.Time, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bans = append(f.bans, banRecord{userID: userID, until: until, ttl: ttl})
	return nil
}

func (f *fakeCounter) GetUserViolationBan(context.Context, int64) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

func (f *fakeCounter) GetUserViolationBans(context.Context, []int64) (map[int64]time.Time, error) {
	return nil, nil
}

func (f *fakeCounter) ClearUserViolationBan(context.Context, int64) error { return nil }

func (f *fakeCounter) banCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bans)
}

type fakeSettings struct{ raw string }

func (f *fakeSettings) GetValue(_ context.Context, key string) (string, error) {
	if key == service.SettingKeyGroupUnavailableAlertEmails {
		return f.raw, nil
	}
	return "", nil
}

type sentMail struct {
	to      string
	subject string
	body    string
}

type fakeEmailSender struct {
	ch chan sentMail
}

func newFakeEmailSender() *fakeEmailSender { return &fakeEmailSender{ch: make(chan sentMail, 8)} }

func (f *fakeEmailSender) SendEmail(_ context.Context, to, subject, body string) error {
	f.ch <- sentMail{to: to, subject: subject, body: body}
	return nil
}

// --- helpers ---

func guardTestConfig(enabled bool, threshold, windowMin, banMin int) securityaudit.ActiveConfig {
	return securityaudit.ActiveConfig{
		Enabled: true,
		Endpoints: []securityaudit.ActiveEndpoint{{
			ID: "guard-1", Name: "Guard", BaseURL: "http://127.0.0.1:8000", Model: "guard-model",
			TimeoutMS: 1000, InputLimit: 1000, Enabled: true,
		}},
		Scanners: []string{"violent"},
		UserGuard: securityaudit.UserGuardConfig{
			Enabled: enabled, Threshold: threshold, WindowMinutes: windowMin, BanDurationMinutes: banMin,
		},
	}
}

func guardTestInput() service.UserGuardCheckInput {
	return service.UserGuardCheckInput{
		Account:   &service.Account{ID: 42, Name: "pool-account", Platform: "anthropic"},
		Protocol:  "anthropic_messages",
		Model:     "claude-test",
		Body:      []byte(`{"model":"claude-test","messages":[{"role":"user","content":"hello"}]}`),
		UserID:    1001,
		Username:  "alice",
		UserEmail: "alice@example.test",
	}
}

func blockDecision() *securityaudit.PromptDecision {
	return &securityaudit.PromptDecision{
		Kind:      securityaudit.DecisionBlock,
		ErrorCode: securityaudit.ErrorCodeBlocked,
		Result: &securityaudit.NormalizedResult{
			Action: securityaudit.ActionBlock, Safety: "Unsafe",
			Categories: []string{"violent"}, MatchedScanners: []string{"violent"},
		},
	}
}

// --- tests ---

func TestCheck_DisabledAllowsWithoutEvaluation(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	counter := &fakeCounter{}
	svc := NewGuardService(&fakeConfigStore{cfg: guardTestConfig(false, 3, 10, 60), active: true}, evaluator, &fakeSettings{}, newFakeEmailSender(), counter)

	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.Nil(t, decision)
	require.Zero(t, evaluator.calls)
	require.Zero(t, counter.incremented)
}

func TestCheck_InactiveConfigAllows(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	svc := NewGuardService(&fakeConfigStore{active: false}, evaluator, &fakeSettings{}, newFakeEmailSender(), &fakeCounter{})

	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.Nil(t, decision)
	require.Zero(t, evaluator.calls)
}

func TestCheck_UnsafeBlocksAndCountsByUser(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	counter := &fakeCounter{}
	svc := NewGuardService(&fakeConfigStore{cfg: guardTestConfig(true, 3, 10, 60), active: true}, evaluator, &fakeSettings{}, newFakeEmailSender(), counter)

	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.True(t, decision.Blocked)
	require.Equal(t, "violent", decision.Reason)
	require.Equal(t, 1, evaluator.calls)
	require.Equal(t, 1, counter.incremented)
	// 未达阈值：不封禁
	require.Zero(t, counter.banCount())
}

func TestCheck_NoUserContextAllowsWithoutCounting(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	counter := &fakeCounter{}
	svc := NewGuardService(&fakeConfigStore{cfg: guardTestConfig(true, 1, 10, 60), active: true}, evaluator, &fakeSettings{}, newFakeEmailSender(), counter)

	input := guardTestInput()
	input.UserID = 0
	decision, err := svc.Check(context.Background(), input)
	require.NoError(t, err)
	require.Nil(t, decision)
	require.Equal(t, 1, evaluator.calls)
	require.Zero(t, counter.incremented)
}

func TestCheck_ThresholdTriggersUserBanResetAndNotify(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	counter := &fakeCounter{claimOK: true}
	email := newFakeEmailSender()
	settings := &fakeSettings{raw: `[{"email":"admin@example.test","verified":true},{"email":"off@example.test","disabled":true,"verified":true}]`}
	svc := NewGuardService(&fakeConfigStore{cfg: guardTestConfig(true, 2, 10, 60), active: true}, evaluator, settings, email, counter)

	// 第一次违规：计数 1，未达阈值
	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.True(t, decision.Blocked)
	require.Zero(t, counter.banCount())

	// 第二次违规：计数 2 达到阈值 → 封禁用户 + 清零 + 邮件
	_, err = svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.Equal(t, 1, counter.banCount())
	ban := counter.bans[0]
	require.Equal(t, int64(1001), ban.userID)
	require.Equal(t, 60*time.Minute, ban.ttl)
	require.Equal(t, 1, counter.reset)

	select {
	case mail := <-email.ch:
		require.Equal(t, "admin@example.test", mail.to)
		require.Contains(t, mail.subject, "alice")
		require.Contains(t, mail.body, "alice@example.test")
		require.Contains(t, mail.body, "violent")
		require.Contains(t, mail.body, "60 分钟")
	case <-time.After(3 * time.Second):
		t.Fatal("expected ban notification email")
	}
	// 被禁用收件人不发
	select {
	case mail := <-email.ch:
		t.Fatalf("unexpected email to %s", mail.to)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestCheck_AdminUserNotBanned(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	counter := &fakeCounter{claimOK: true}
	email := newFakeEmailSender()
	settings := &fakeSettings{raw: `[{"email":"admin@example.test","verified":true}]`}
	svc := NewGuardService(&fakeConfigStore{cfg: guardTestConfig(true, 1, 10, 60), active: true}, evaluator, settings, email, counter)

	input := guardTestInput()
	input.UserIsAdmin = true
	decision, err := svc.Check(context.Background(), input)
	require.NoError(t, err)
	// 违规请求本身仍被 403 阻断并计数，但不封禁管理员
	require.True(t, decision.Blocked)
	require.Equal(t, 1, counter.incremented)
	require.Zero(t, counter.banCount())
}

func TestCheck_EvaluatorErrorFailsOpen(t *testing.T) {
	evaluator := &fakeEvaluator{err: errors.New("guard endpoint timeout")}
	counter := &fakeCounter{}
	svc := NewGuardService(&fakeConfigStore{cfg: guardTestConfig(true, 1, 10, 60), active: true}, evaluator, &fakeSettings{}, newFakeEmailSender(), counter)

	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.Nil(t, decision)
	require.Equal(t, 1, evaluator.calls)
	require.Zero(t, counter.incremented)
}

func TestCheck_FlagDecisionAllowsWithoutCounting(t *testing.T) {
	// Controversial / 未命中已启用分类的 Unsafe → DecisionFlag，不计数
	evaluator := &fakeEvaluator{decision: &securityaudit.PromptDecision{Kind: securityaudit.DecisionFlag, AllowNextStage: true}}
	counter := &fakeCounter{}
	svc := NewGuardService(&fakeConfigStore{cfg: guardTestConfig(true, 1, 10, 60), active: true}, evaluator, &fakeSettings{}, newFakeEmailSender(), counter)

	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.Nil(t, decision)
	require.Zero(t, counter.incremented)
}

func TestNotify_CooldownSuppressesEmail(t *testing.T) {
	counter := &fakeCounter{claimOK: false}
	email := newFakeEmailSender()
	settings := &fakeSettings{raw: `[{"email":"admin@example.test","verified":true}]`}
	svc := NewGuardService(&fakeConfigStore{active: false}, nil, settings, email, counter)

	err := svc.Notify(context.Background(), ViolationNotifyInput{UserID: 1001, Username: "alice", Reason: "violent"})
	require.NoError(t, err)
	select {
	case mail := <-email.ch:
		t.Fatalf("cooldown should suppress email, got %s", mail.to)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestNotify_NoRecipientsSkips(t *testing.T) {
	counter := &fakeCounter{claimOK: true}
	email := newFakeEmailSender()
	svc := NewGuardService(&fakeConfigStore{active: false}, nil, &fakeSettings{raw: "[]"}, email, counter)

	err := svc.Notify(context.Background(), ViolationNotifyInput{UserID: 1001, Reason: "violent"})
	require.NoError(t, err)
	select {
	case mail := <-email.ch:
		t.Fatalf("no recipients should skip email, got %s", mail.to)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestCheck_WhitelistedUserSkipsEvaluation(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	counter := &fakeCounter{}
	cfg := guardTestConfig(true, 1, 10, 60)
	cfg.UserGuard.WhitelistUserIDs = []int64{7, 1001, 2000}
	svc := NewGuardService(&fakeConfigStore{cfg: cfg, active: true}, evaluator, &fakeSettings{}, newFakeEmailSender(), counter)

	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.Nil(t, decision)
	// 白名单用户不产生任何审核 API 调用、不计数
	require.Zero(t, evaluator.calls)
	require.Zero(t, counter.incremented)
	require.Zero(t, counter.banCount())
}

func TestCheck_NonWhitelistedUserStillAudited(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	counter := &fakeCounter{}
	cfg := guardTestConfig(true, 1, 10, 60)
	cfg.UserGuard.WhitelistUserIDs = []int64{7, 2000}
	svc := NewGuardService(&fakeConfigStore{cfg: cfg, active: true}, evaluator, &fakeSettings{}, newFakeEmailSender(), counter)

	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.True(t, decision.Blocked)
	require.Equal(t, 1, evaluator.calls)
	require.Equal(t, 1, counter.incremented)
}

func TestCheck_EmptyWhitelistBehaviorUnchanged(t *testing.T) {
	evaluator := &fakeEvaluator{decision: blockDecision()}
	counter := &fakeCounter{}
	cfg := guardTestConfig(true, 1, 10, 60)
	cfg.UserGuard.WhitelistUserIDs = []int64{}
	svc := NewGuardService(&fakeConfigStore{cfg: cfg, active: true}, evaluator, &fakeSettings{}, newFakeEmailSender(), counter)

	decision, err := svc.Check(context.Background(), guardTestInput())
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.True(t, decision.Blocked)
	require.Equal(t, 1, evaluator.calls)
	require.Equal(t, 1, counter.incremented)
}
