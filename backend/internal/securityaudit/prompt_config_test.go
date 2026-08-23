package securityaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type prefixEncryptor struct{}

func (prefixEncryptor) Encrypt(value string) (string, error) { return "enc:" + value, nil }
func (prefixEncryptor) Decrypt(value string) (string, error) {
	if !strings.HasPrefix(value, "enc:") {
		return "", errors.New("cipher: message authentication failed")
	}
	return value[4:], nil
}

func boolPtr(value bool) *bool { return &value }

func intPtr(value int) *int { return &value }

// testTotpKeyConfig mirrors a deployment with a fixed TOTP_ENCRYPTION_KEY so
// unit tests may persist endpoint tokens.
func testTotpKeyConfig() *config.Config {
	return &config.Config{Totp: config.TotpConfig{EncryptionKeyConfigured: true}}
}

func TestDefaultConfigIsOff(t *testing.T) {
	storage, err := ParseStorageConfig("")
	require.NoError(t, err)
	require.False(t, storage.Enabled)
	require.False(t, storage.BlockingLatestTurnOnly)
	// AsyncLatestTurnOnly 默认 true：异步审计只提取最新一条 user 消息。
	require.True(t, storage.AsyncLatestTurnOnly)
	active, err := ActiveFromStorage(storage, true, prefixEncryptor{})
	require.NoError(t, err)
	require.Equal(t, ModeOff, active.EffectiveMode())
	require.Equal(t, AllScannerIDs, storage.Scanners)
	require.Equal(t, []string{"user"}, storage.AuditRoles)
	publicJSON, err := json.Marshal(PublicFromStorage(storage, true, nil))
	require.NoError(t, err)
	require.Contains(t, string(publicJSON), `"group_ids":[]`)
	require.Contains(t, string(publicJSON), `"endpoints":[]`)
}

func TestBlockingLatestTurnOnlyConfigRoundTrip(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}
	request := UpdateConfigRequest{
		ExpectedConfigVersion: 1, Enabled: true, BlockingEnabled: true, BlockingLatestTurnOnly: true,
		Strategy: "priority", WorkerCount: 1, QueueCapacity: 10, Scanners: []string{"pii"}, AllGroups: true,
		Endpoints: []UpdateEndpoint{{
			ID: "guard-1", Name: "Guard", Protocol: "openai_compatible", BaseURL: "http://127.0.0.1:8080",
			Model: DefaultGuardModel, TimeoutMS: 1000, InputLimit: 1000, Enabled: true,
		}},
	}
	next, err := manager.buildNextStorage(DefaultStorageConfig(), request, 9)
	require.NoError(t, err)
	require.True(t, next.BlockingLatestTurnOnly)
	require.Contains(t, changeSummary(next), `"blocking_latest_turn_only":true`)

	active, err := ActiveFromStorage(next, true, prefixEncryptor{})
	require.NoError(t, err)
	require.True(t, active.BlockingLatestTurnOnly)
	public := PublicFromStorage(next, true, nil)
	require.True(t, public.BlockingLatestTurnOnly)
}

func TestAsyncLatestTurnOnlyConfigRoundTrip(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}
	// update 携带 true：storage/active/public 全载体透传，changeSummary 落字段。
	request := promptAuditUpdateRequest(1, 1, "")
	request.AsyncLatestTurnOnly = boolPtr(true)
	next, err := manager.buildNextStorage(DefaultStorageConfig(), request, 9)
	require.NoError(t, err)
	require.True(t, next.AsyncLatestTurnOnly)
	require.Contains(t, changeSummary(next), `"async_latest_turn_only":true`)

	active, err := ActiveFromStorage(next, true, prefixEncryptor{})
	require.NoError(t, err)
	require.True(t, active.AsyncLatestTurnOnly)
	public := PublicFromStorage(next, true, nil)
	require.True(t, public.AsyncLatestTurnOnly)

	raw, err := json.Marshal(next)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"async_latest_turn_only":true`)
	parsed, err := ParseStorageConfig(string(raw))
	require.NoError(t, err)
	require.True(t, parsed.AsyncLatestTurnOnly)

	// 显式 false 也可完整往返（更新路径允许关闭收窄，恢复多轮 user 历史）。
	request.AsyncLatestTurnOnly = boolPtr(false)
	nextOff, err := manager.buildNextStorage(parsed, request, 9)
	require.NoError(t, err)
	require.False(t, nextOff.AsyncLatestTurnOnly)
	activeOff, err := ActiveFromStorage(nextOff, true, prefixEncryptor{})
	require.NoError(t, err)
	require.False(t, activeOff.AsyncLatestTurnOnly)
	require.False(t, PublicFromStorage(nextOff, true, nil).AsyncLatestTurnOnly)

	// 旧配置缺省该字段时按默认 true 生效；显式 false 的持久化配置保持 false。
	legacy, err := ParseStorageConfig(`{"enabled":false,"config_version":9}`)
	require.NoError(t, err)
	require.True(t, legacy.AsyncLatestTurnOnly)
	off, err := ParseStorageConfig(`{"enabled":false,"async_latest_turn_only":false,"config_version":9}`)
	require.NoError(t, err)
	require.False(t, off.AsyncLatestTurnOnly)
}

func TestAsyncLatestTurnOnlyUpdateOmitsFieldPreservesStorage(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}

	// update 省略 async_latest_turn_only（nil）：保留存储现值，不静默重置为 false。
	// 覆盖存储为 true 与 false 两条，防止 UI 保存把配置打回默认/关掉收窄。
	for _, stored := range []bool{true, false} {
		base := DefaultStorageConfig()
		base.AsyncLatestTurnOnly = stored
		request := promptAuditUpdateRequest(1, 1, "")
		next, err := manager.buildNextStorage(base, request, 9)
		require.NoError(t, err)
		require.Equal(t, stored, next.AsyncLatestTurnOnly)
	}

	// 显式传 true/false 时正确覆盖存储值（双向）。
	for _, override := range []bool{true, false} {
		base := DefaultStorageConfig()
		base.AsyncLatestTurnOnly = !override
		request := promptAuditUpdateRequest(1, 1, "")
		request.AsyncLatestTurnOnly = boolPtr(override)
		next, err := manager.buildNextStorage(base, request, 9)
		require.NoError(t, err)
		require.Equal(t, override, next.AsyncLatestTurnOnly)
	}
}

func TestConfigRejectsBlockingWithoutAudit(t *testing.T) {
	storage := DefaultStorageConfig()
	storage.BlockingEnabled = true
	require.Error(t, validateStorageConfig(storage))
}

func TestPublicConfigNeverMarshalsToken(t *testing.T) {
	storage := DefaultStorageConfig()
	storage.Endpoints = []StorageEndpoint{{ID: "one", Name: "One", Protocol: "openai_compatible", BaseURL: "http://127.0.0.1:8080", Model: DefaultGuardModel, TokenCiphertext: "GUARD_TOKEN_CANARY_SECRET", TimeoutMS: 1000, InputLimit: 1000, Enabled: true}}
	public := PublicFromStorage(storage, true, nil)
	raw, err := json.Marshal(public)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "GUARD_TOKEN_CANARY_SECRET")
	require.NotContains(t, string(raw), "ciphertext")
	require.True(t, public.Endpoints[0].HasToken)
}

func TestConfigRuntimeLoadErrorIsStableBoundedAndSecretFree(t *testing.T) {
	const canary = "CONFIG_LOAD_CANARY_SECRET"
	manager := &ConfigManager{clock: fixedClock{}}
	manager.recordLoadError(errors.New("decrypt failed for token " + canary + " Authorization: Bearer " + canary))
	_, _, _, message := manager.RuntimeState()
	require.Equal(t, stableErrorMessage("config_load_failed"), message)
	require.NotContains(t, message, canary)
	require.LessOrEqual(t, len([]rune(message)), 160)
}

func TestConfigManagerPublicRequiresSuccessfullyLoadedSnapshot(t *testing.T) {
	t.Run("absent persisted setting is legitimate default", func(t *testing.T) {
		manager := NewConfigManager(nil, staticSettingRepository{values: map[string]string{
			SettingKeyPromptAuditConfig: "",
			SettingKeyRiskControl:       "false",
		}}, nil, prefixEncryptor{}, testTotpKeyConfig())
		require.NoError(t, manager.Reload(context.Background()))

		public, err := manager.Public()
		require.NoError(t, err)
		require.Equal(t, int64(1), public.ConfigVersion)
		require.False(t, public.Enabled)
	})

	t.Run("unparseable persisted config is unavailable", func(t *testing.T) {
		const canary = "persisted-token-canary"
		manager := NewConfigManager(nil, staticSettingRepository{values: map[string]string{
			// Endpoint without id/name fails validation, so no trustworthy
			// snapshot can be installed from this raw value.
			SettingKeyPromptAuditConfig: `{"enabled":true,"config_version":9,"endpoints":[{"token_ciphertext":"` + canary + `"}]}`,
			SettingKeyRiskControl:       "true",
		}}, nil, prefixEncryptor{}, testTotpKeyConfig())
		require.Error(t, manager.Reload(context.Background()))

		public, err := manager.Public()
		require.Error(t, err)
		require.Empty(t, public)
		require.Equal(t, ErrorCodeConfigUnavailable, infraerrors.Reason(err))
		require.NotContains(t, err.Error(), canary)
	})

	t.Run("reload failure preserves last successfully loaded snapshot", func(t *testing.T) {
		storage := DefaultStorageConfig()
		storage.ConfigVersion = 4
		storage.ChangeSummary = "trusted snapshot"
		raw, err := json.Marshal(storage)
		require.NoError(t, err)
		repository := &switchableSettingRepository{staticSettingRepository: staticSettingRepository{values: map[string]string{
			SettingKeyPromptAuditConfig: string(raw),
			SettingKeyRiskControl:       "false",
		}}}
		manager := NewConfigManager(nil, repository, nil, prefixEncryptor{}, testTotpKeyConfig())
		require.NoError(t, manager.Reload(context.Background()))
		repository.loadErr = errors.New("settings unavailable")
		require.Error(t, manager.Reload(context.Background()))

		public, err := manager.Public()
		require.NoError(t, err)
		require.Equal(t, int64(4), public.ConfigVersion)
		require.Equal(t, "trusted snapshot", public.ChangeSummary)
	})
}

// Regression coverage for issue #4887: a persisted config whose endpoint token
// can no longer be decrypted (encryption key changed or auto-generated per
// boot) must stay visible and editable for admins instead of falling back to a
// default v1 config that makes every save fail the CAS version check.
func TestConfigManagerUndecryptableTokenKeepsConfigVisibleAndRecoverable(t *testing.T) {
	const canary = "persisted-token-canary"
	persisted := `{"enabled":true,"blocking_enabled":false,"config_version":9,"endpoints":[{"id":"g1","name":"Guard","protocol":"openai_compatible","base_url":"http://127.0.0.1:8080","model":"m","token_ciphertext":"` + canary + `","timeout_ms":1000,"input_limit":1000,"enabled":true}]}`
	manager := NewConfigManager(nil, staticSettingRepository{values: map[string]string{
		SettingKeyPromptAuditConfig: persisted,
		SettingKeyRiskControl:       "true",
	}}, nil, prefixEncryptor{}, testTotpKeyConfig())
	require.NoError(t, manager.Reload(context.Background()), "an undecryptable token must not fail the whole config load")

	public, err := manager.Public()
	require.NoError(t, err)
	require.Equal(t, int64(9), public.ConfigVersion, "admins must see the real persisted version so CAS saves can succeed")
	require.Len(t, public.Endpoints, 1)
	require.True(t, public.Endpoints[0].HasToken)
	require.Equal(t, "invalid", public.Endpoints[0].TokenStatus)
	raw, err := json.Marshal(public)
	require.NoError(t, err)
	require.NotContains(t, string(raw), canary)

	active, ok := manager.Active()
	require.True(t, ok)
	require.Len(t, active.Endpoints, 1)
	require.False(t, active.Endpoints[0].Enabled, "an endpoint with an undecryptable token must not be used at runtime")
	require.True(t, active.Endpoints[0].TokenInvalid)
	require.Empty(t, active.Endpoints[0].Token)
	require.Empty(t, active.EnabledEndpoints())
	require.Equal(t, []string{"g1"}, active.InvalidTokenEndpointIDs())

	expected, activeVersion, _, _ := manager.RuntimeState()
	require.Equal(t, int64(9), expected)
	require.Equal(t, int64(9), activeVersion)
}

func TestConfigManagerUndecryptableTokenStillFailsClosedForBlockingIntent(t *testing.T) {
	persisted := `{"enabled":true,"blocking_enabled":true,"config_version":9,"endpoints":[{"id":"g1","name":"Guard","protocol":"openai_compatible","base_url":"http://127.0.0.1:8080","model":"m","token_ciphertext":"undecryptable","timeout_ms":1000,"input_limit":1000,"enabled":true}]}`
	manager := NewConfigManager(nil, staticSettingRepository{values: map[string]string{
		SettingKeyPromptAuditConfig: persisted,
		SettingKeyRiskControl:       "true",
	}}, nil, prefixEncryptor{}, testTotpKeyConfig())
	require.NoError(t, manager.Reload(context.Background()))
	require.Equal(t, ModeBlocking, manager.EffectiveMode())

	service := &PromptService{config: manager, evaluator: NewGuardEvaluator(NewOpenAICompatibleScanner(), nil, nil)}
	decision, err := service.Evaluate(context.Background(), Request{
		Protocol: "openai_chat_completions",
		Body:     []byte(`{"messages":[{"role":"user","content":"hi"}]}`),
	})
	require.Error(t, err, "blocking intent with no usable endpoint must not let requests pass unaudited")
	require.Nil(t, decision)
	var guardErr *GuardError
	require.ErrorAs(t, err, &guardErr)
	require.Equal(t, ErrorCodeUnavailable, guardErr.Code)
}

func TestBuildNextStoragePreserveReplaceAndClearToken(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}
	current := DefaultStorageConfig()
	current.Endpoints = []StorageEndpoint{{ID: "one", Name: "One", Protocol: "openai_compatible", BaseURL: "http://127.0.0.1:8080", Model: DefaultGuardModel, TokenCiphertext: "enc:old", TimeoutMS: 1000, InputLimit: 1000}}
	base := UpdateConfigRequest{ExpectedConfigVersion: 1, Strategy: "priority", WorkerCount: 1, QueueCapacity: 10, Scanners: []string{"PII"}, AllGroups: true,
		Endpoints: []UpdateEndpoint{{ID: "one", Name: "One", Protocol: "openai_compatible", BaseURL: "http://127.0.0.1:8080", TimeoutMS: 1000, InputLimit: 1000}}}
	preserved, err := manager.buildNextStorage(current, base, 9)
	require.NoError(t, err)
	require.Equal(t, "enc:old", preserved.Endpoints[0].TokenCiphertext)
	replacedReq := base
	replacedReq.Endpoints = append([]UpdateEndpoint(nil), base.Endpoints...)
	replacedReq.Endpoints[0].Token = "new"
	replaced, err := manager.buildNextStorage(current, replacedReq, 9)
	require.NoError(t, err)
	require.Equal(t, "enc:new", replaced.Endpoints[0].TokenCiphertext)
	clearedReq := base
	clearedReq.Endpoints = append([]UpdateEndpoint(nil), base.Endpoints...)
	clearedReq.Endpoints[0].ClearToken = true
	cleared, err := manager.buildNextStorage(current, clearedReq, 9)
	require.NoError(t, err)
	require.Empty(t, cleared.Endpoints[0].TokenCiphertext)
}

// Without a fixed encryption key the per-boot auto-generated key would make a
// freshly saved token undecryptable after the next restart (issue #4887), so
// saving a new token must be rejected with an actionable error. Preserving or
// clearing an existing ciphertext stays allowed so admins can still edit or
// disable the feature.
func TestBuildNextStorageRejectsNewTokenWithoutConfiguredEncryptionKey(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: false}
	current := DefaultStorageConfig()
	current.Endpoints = []StorageEndpoint{{ID: "one", Name: "One", Protocol: "openai_compatible", BaseURL: "http://127.0.0.1:8080", Model: DefaultGuardModel, TokenCiphertext: "enc:old", TimeoutMS: 1000, InputLimit: 1000}}
	base := UpdateConfigRequest{ExpectedConfigVersion: 1, Strategy: "priority", WorkerCount: 1, QueueCapacity: 10, Scanners: []string{"PII"}, AllGroups: true,
		Endpoints: []UpdateEndpoint{{ID: "one", Name: "One", Protocol: "openai_compatible", BaseURL: "http://127.0.0.1:8080", TimeoutMS: 1000, InputLimit: 1000}}}

	newTokenReq := base
	newTokenReq.Endpoints = append([]UpdateEndpoint(nil), base.Endpoints...)
	newTokenReq.Endpoints[0].Token = "fresh-token"
	_, err := manager.buildNextStorage(current, newTokenReq, 9)
	require.Error(t, err)
	require.Equal(t, ErrorCodeEncryptionKeyRequired, infraerrors.Reason(err))

	preserved, err := manager.buildNextStorage(current, base, 9)
	require.NoError(t, err)
	require.Equal(t, "enc:old", preserved.Endpoints[0].TokenCiphertext)

	clearedReq := base
	clearedReq.Endpoints = append([]UpdateEndpoint(nil), base.Endpoints...)
	clearedReq.Endpoints[0].ClearToken = true
	cleared, err := manager.buildNextStorage(current, clearedReq, 9)
	require.NoError(t, err)
	require.Empty(t, cleared.Endpoints[0].TokenCiphertext)
}

func TestEffectiveModeTruthTable(t *testing.T) {
	tests := []struct {
		risk, enabled, blocking bool
		want                    Mode
	}{
		{false, false, false, ModeOff}, {false, true, true, ModeOff}, {true, false, false, ModeOff},
		{true, true, false, ModeAsync}, {true, true, true, ModeBlocking},
	}
	for _, tt := range tests {
		cfg := ActiveConfig{RiskControlEnabled: tt.risk, Enabled: tt.enabled, BlockingEnabled: tt.blocking}
		require.Equal(t, tt.want, cfg.EffectiveMode())
	}
}

func TestConfigManagerColdStartOnlyFailsClosedForExplicitBlockingIntent(t *testing.T) {
	manager := &ConfigManager{}

	manager.observeExpectedState(`{"enabled":true,"blocking_enabled":false,"config_version":42}`, true)
	require.Equal(t, int64(42), manager.expected.Load())
	require.Equal(t, ModeOff, manager.EffectiveMode(), "an async config version must not imply blocking")
	require.False(t, manager.BlockingActivationDegraded())

	manager.observeExpectedState(`{"enabled":true,"blocking_enabled":true,"config_version":43}`, false)
	require.Equal(t, ModeOff, manager.EffectiveMode(), "the global risk-control switch still gates blocking")

	manager.observeExpectedState(`{"enabled":true,"blocking_enabled":true,"config_version":44}`, true)
	require.Equal(t, ModeBlocking, manager.EffectiveMode())
	require.True(t, manager.BlockingActivationDegraded())

	manager.observeExpectedState(`{"enabled":true`, true)
	require.Equal(t, ModeBlocking, manager.EffectiveMode(), "undecodable storage must not erase the last known strict intent")
}

func TestConfigManagerStaleWeakerSnapshotFailsClosedWhenBlockingExpected(t *testing.T) {
	manager := &ConfigManager{}
	async := ActiveConfig{RiskControlEnabled: true, Enabled: true, BlockingEnabled: false, ConfigVersion: 1}
	manager.snapshot.Store(&activeConfigSnapshot{active: async, storage: DefaultStorageConfig(), loadedAt: fixedClock{}.Now()})
	manager.expected.Store(2)
	manager.expectedBlocking.Store(true)

	require.True(t, manager.BlockingActivationDegraded())
	require.Equal(t, ModeBlocking, manager.EffectiveMode())

	service := &PromptService{config: manager, evaluator: NewGuardEvaluator(nil, nil, nil)}
	decision, err := service.Evaluate(context.Background(), Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"user","content":"hi"}]}`)})
	require.Error(t, err)
	require.Nil(t, decision)
	var guardErr *GuardError
	require.ErrorAs(t, err, &guardErr)
	require.Equal(t, ErrorCodeUnavailable, guardErr.Code)
}

type errorSettingRepository struct{ staticSettingRepository }

func (errorSettingRepository) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, errors.New("settings unavailable")
}

type switchableSettingRepository struct {
	staticSettingRepository
	loadErr error
}

func (r *switchableSettingRepository) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r.staticSettingRepository.GetMultiple(ctx, keys)
}

func TestConfigManagerStartupLoadFailureDoesNotBlockWhenBlockingNotIntended(t *testing.T) {
	// Settings unavailable and no prior blocking intent: stay ModeOff so the
	// gateway remains usable and admins can still disable/configure Prompt Audit.
	manager := NewConfigManager(nil, errorSettingRepository{}, nil, prefixEncryptor{}, testTotpKeyConfig())
	err := manager.Start(context.Background())
	require.Error(t, err)
	require.True(t, manager.configUntrusted.Load())
	require.False(t, manager.BlockingActivationDegraded())
	require.Equal(t, ModeOff, manager.EffectiveMode())

	service := &PromptService{config: manager, evaluator: NewGuardEvaluator(nil, nil, nil)}
	decision, evalErr := service.Evaluate(context.Background(), Request{
		Protocol: "openai_chat_completions",
		Body:     []byte(`{"messages":[{"role":"user","content":"hi"}]}`),
	})
	require.NoError(t, evalErr)
	require.NotNil(t, decision)
	require.Equal(t, DecisionAllow, decision.Kind)
	require.NoError(t, manager.Shutdown(context.Background()))
}

func TestConfigManagerStartupLoadFailureFailsClosedWhenBlockingIntended(t *testing.T) {
	manager := NewConfigManager(nil, errorSettingRepository{}, nil, prefixEncryptor{}, testTotpKeyConfig())
	// Simulate intent observed before a later load failure (e.g. decrypt error).
	manager.observeExpectedState(`{"enabled":true,"blocking_enabled":true,"config_version":3}`, true)
	manager.markConfigUntrusted()
	require.True(t, manager.BlockingActivationDegraded())
	require.Equal(t, ModeBlocking, manager.EffectiveMode())

	service := &PromptService{config: manager, evaluator: NewGuardEvaluator(nil, nil, nil)}
	decision, err := service.Evaluate(context.Background(), Request{
		Protocol: "openai_chat_completions",
		Body:     []byte(`{"messages":[{"role":"user","content":"hi"}]}`),
	})
	require.Error(t, err)
	require.Nil(t, decision)
	var guardErr *GuardError
	require.ErrorAs(t, err, &guardErr)
	require.Equal(t, ErrorCodeUnavailable, guardErr.Code)
}

func TestConfigManagerUntrustedClearsOnSuccessfulDisable(t *testing.T) {
	// After a degraded fail-closed period, saving disabled config must restore ModeOff.
	manager := &ConfigManager{encryptor: prefixEncryptor{}, clock: fixedClock{}}
	manager.observeExpectedState(`{"enabled":true,"blocking_enabled":true,"config_version":5}`, true)
	manager.markConfigUntrusted()
	require.Equal(t, ModeBlocking, manager.EffectiveMode())

	// Install a trusted disabled snapshot the same way Save does after commit.
	disabled := DefaultStorageConfig()
	disabled.ConfigVersion = 6
	disabled.Enabled = false
	disabled.BlockingEnabled = false
	active, err := ActiveFromStorage(disabled, true, manager.encryptor)
	require.NoError(t, err)
	manager.expected.Store(disabled.ConfigVersion)
	manager.expectedBlocking.Store(false)
	manager.snapshot.Store(&activeConfigSnapshot{storage: disabled, active: active, loadedAt: manager.clock.Now()})
	manager.configUntrusted.Store(false)

	require.False(t, manager.BlockingActivationDegraded())
	require.Equal(t, ModeOff, manager.EffectiveMode())

	service := &PromptService{config: manager, evaluator: NewGuardEvaluator(nil, nil, nil)}
	decision, evalErr := service.Evaluate(context.Background(), Request{
		Protocol: "openai_chat_completions",
		Body:     []byte(`{"messages":[{"role":"user","content":"hi"}]}`),
	})
	require.NoError(t, evalErr)
	require.Equal(t, DecisionAllow, decision.Kind)
}

func TestConfigManagerUntrustedWithoutBlockingDoesNotForceBlockingMode(t *testing.T) {
	manager := &ConfigManager{}
	manager.observeExpectedState(`{"enabled":true,"blocking_enabled":false,"config_version":2}`, true)
	manager.markConfigUntrusted()
	require.False(t, manager.expectedBlocking.Load())
	require.False(t, manager.BlockingActivationDegraded())
	require.Equal(t, ModeOff, manager.EffectiveMode(), "async intent + untrusted must not force blocking unavailable")
}

func TestParseLegacyConfigDefaultsMissingFieldsWithoutEnablingBlocking(t *testing.T) {
	storage, err := ParseStorageConfig(`{"enabled":false,"config_version":9}`)
	require.NoError(t, err)
	require.False(t, storage.BlockingEnabled)
	require.Equal(t, "priority", storage.Strategy)
	require.Equal(t, DefaultWorkerCount, storage.WorkerCount)
	require.Equal(t, DefaultQueueCapacity, storage.QueueCapacity)
	require.Equal(t, AllScannerIDs, storage.Scanners)
	require.Equal(t, []string{"user"}, storage.AuditRoles)
	require.True(t, storage.AllGroups)
	require.True(t, storage.AsyncLatestTurnOnly)
}

func TestUpdateConfigStrictBoundsAndKnownValues(t *testing.T) {
	valid := promptAuditUpdateRequest(1, 1, "")
	require.NoError(t, validateUpdateConfigRequest(valid))

	tests := []struct {
		name   string
		mutate func(*UpdateConfigRequest)
		reason string
	}{
		{name: "strategy", mutate: func(req *UpdateConfigRequest) { req.Strategy = "round_robin" }, reason: "prompt_audit_invalid_strategy"},
		{name: "worker low", mutate: func(req *UpdateConfigRequest) { req.WorkerCount = 0 }, reason: "prompt_audit_invalid_worker_count"},
		{name: "worker high", mutate: func(req *UpdateConfigRequest) { req.WorkerCount = MaxWorkerCount + 1 }, reason: "prompt_audit_invalid_worker_count"},
		{name: "capacity low", mutate: func(req *UpdateConfigRequest) { req.QueueCapacity = 0 }, reason: "prompt_audit_invalid_queue_capacity"},
		{name: "capacity high", mutate: func(req *UpdateConfigRequest) { req.QueueCapacity = MaxQueueCapacity + 1 }, reason: "prompt_audit_invalid_queue_capacity"},
		{name: "unknown scanner", mutate: func(req *UpdateConfigRequest) { req.Scanners = []string{"made_up"} }, reason: "prompt_audit_invalid_scanner"},
		{name: "group required", mutate: func(req *UpdateConfigRequest) { req.AllGroups = false; req.GroupIDs = nil }, reason: "prompt_audit_groups_required"},
		{name: "group positive", mutate: func(req *UpdateConfigRequest) { req.AllGroups = false; req.GroupIDs = []int64{0} }, reason: "prompt_audit_invalid_group"},
		{name: "timeout low", mutate: func(req *UpdateConfigRequest) { req.Endpoints[0].TimeoutMS = MinTimeoutMS - 1 }, reason: "prompt_audit_invalid_timeout"},
		{name: "timeout high", mutate: func(req *UpdateConfigRequest) { req.Endpoints[0].TimeoutMS = MaxTimeoutMS + 1 }, reason: "prompt_audit_invalid_timeout"},
		{name: "input low", mutate: func(req *UpdateConfigRequest) { req.Endpoints[0].InputLimit = MinInputLimit - 1 }, reason: "prompt_audit_invalid_input_limit"},
		{name: "input high", mutate: func(req *UpdateConfigRequest) { req.Endpoints[0].InputLimit = MaxInputLimit + 1 }, reason: "prompt_audit_invalid_input_limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			req.Scanners = append([]string(nil), valid.Scanners...)
			req.GroupIDs = append([]int64(nil), valid.GroupIDs...)
			req.Endpoints = append([]UpdateEndpoint(nil), valid.Endpoints...)
			tt.mutate(&req)
			err := validateUpdateConfigRequest(req)
			require.Error(t, err)
			require.Equal(t, tt.reason, infraerrors.Reason(err))
		})
	}
}

func TestUserGuardConfigValidation(t *testing.T) {
	base := DefaultStorageConfig()
	base.Endpoints = []StorageEndpoint{{
		ID: "guard-1", Name: "Guard", Protocol: "openai_compatible", BaseURL: "http://127.0.0.1:8080",
		Model: DefaultGuardModel, TimeoutMS: 1000, InputLimit: 1000, Enabled: true,
	}}

	// 未启用时零值合法
	require.NoError(t, validateStorageConfig(base))

	valid := base
	valid.UserGuard = UserGuardConfig{Enabled: true, Threshold: 3, WindowMinutes: 10, BanDurationMinutes: 60}
	require.NoError(t, validateStorageConfig(valid))

	tests := []struct {
		name   string
		guard  UserGuardConfig
		reason string
	}{
		{name: "threshold low", guard: UserGuardConfig{Enabled: true, Threshold: 0, WindowMinutes: 10, BanDurationMinutes: 60}, reason: "user_guard_invalid_threshold"},
		{name: "threshold high", guard: UserGuardConfig{Enabled: true, Threshold: 101, WindowMinutes: 10, BanDurationMinutes: 60}, reason: "user_guard_invalid_threshold"},
		{name: "window low", guard: UserGuardConfig{Enabled: true, Threshold: 3, WindowMinutes: 0, BanDurationMinutes: 60}, reason: "user_guard_invalid_window"},
		{name: "window high", guard: UserGuardConfig{Enabled: true, Threshold: 3, WindowMinutes: 1441, BanDurationMinutes: 60}, reason: "user_guard_invalid_window"},
		{name: "ban low", guard: UserGuardConfig{Enabled: true, Threshold: 3, WindowMinutes: 10, BanDurationMinutes: 0}, reason: "user_guard_invalid_ban_duration"},
		{name: "ban high", guard: UserGuardConfig{Enabled: true, Threshold: 3, WindowMinutes: 10, BanDurationMinutes: 10081}, reason: "user_guard_invalid_ban_duration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.UserGuard = tt.guard
			err := validateStorageConfig(cfg)
			require.Error(t, err)
			require.Equal(t, tt.reason, infraerrors.Reason(err))

			req := promptAuditUpdateRequest(1, 1, "")
			req.UserGuard = tt.guard
			reqErr := validateUpdateConfigRequest(req)
			require.Error(t, reqErr)
			require.Equal(t, tt.reason, infraerrors.Reason(reqErr))
		})
	}

	// 启用守护但审计本身未启用、且无启用节点时拒绝
	noEndpoint := DefaultStorageConfig()
	noEndpoint.UserGuard = UserGuardConfig{Enabled: true, Threshold: 3, WindowMinutes: 10, BanDurationMinutes: 60}
	err := validateStorageConfig(noEndpoint)
	require.Error(t, err)
	require.Equal(t, "prompt_audit_endpoint_required", infraerrors.Reason(err))
}

func TestUserGuardConfigRoundTrip(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}
	request := promptAuditUpdateRequest(1, 1, "")
	request.UserGuard = UserGuardConfig{Enabled: true, Threshold: 5, WindowMinutes: 30, BanDurationMinutes: 120}

	next, err := manager.buildNextStorage(DefaultStorageConfig(), request, 9)
	require.NoError(t, err)
	require.Equal(t, UserGuardConfig{Enabled: true, Threshold: 5, WindowMinutes: 30, BanDurationMinutes: 120, WhitelistUserIDs: []int64{}}, next.UserGuard)

	active, err := ActiveFromStorage(next, true, prefixEncryptor{})
	require.NoError(t, err)
	require.Equal(t, next.UserGuard, active.UserGuard)

	public := PublicFromStorage(next, true, nil)
	require.Equal(t, next.UserGuard, public.UserGuard)

	raw, err := json.Marshal(next)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"user_guard"`)
	parsed, err := ParseStorageConfig(string(raw))
	require.NoError(t, err)
	require.Equal(t, next.UserGuard, parsed.UserGuard)
}

func TestAuditRolesConfigRoundTripAndValidation(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}
	request := promptAuditUpdateRequest(1, 1, "")
	request.AuditRoles = []string{"user", "system", "assistant"}
	next, err := manager.buildNextStorage(DefaultStorageConfig(), request, 9)
	require.NoError(t, err)
	require.Equal(t, []string{"user", "system", "assistant"}, next.AuditRoles)

	active, err := ActiveFromStorage(next, true, prefixEncryptor{})
	require.NoError(t, err)
	require.Equal(t, []string{"user", "system", "assistant"}, active.AuditRoles)

	public := PublicFromStorage(next, true, nil)
	require.Equal(t, []string{"user", "system", "assistant"}, public.AuditRoles)

	raw, err := json.Marshal(next)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"audit_roles":["user","system","assistant"]`)
	parsed, err := ParseStorageConfig(string(raw))
	require.NoError(t, err)
	require.Equal(t, []string{"user", "system", "assistant"}, parsed.AuditRoles)

	// 旧配置缺省该字段时按默认 ["user"] 生效；空列表同样回退默认。
	legacy, err := ParseStorageConfig(`{"enabled":false,"config_version":9}`)
	require.NoError(t, err)
	require.Equal(t, []string{"user"}, legacy.AuditRoles)
	empty, err := ParseStorageConfig(`{"audit_roles":[]}`)
	require.NoError(t, err)
	require.Equal(t, []string{"user"}, empty.AuditRoles)

	// 存储解析路径：未知角色归一化丢弃（与 scanner 行为一致），大小写/重复归并；
	// model 是 gemini 协议合法角色，可经配置保留（修复前被误当未知角色丢弃）。
	dropped, err := ParseStorageConfig(`{"audit_roles":["user","model","SYSTEM","user","evil"]}`)
	require.NoError(t, err)
	require.Equal(t, []string{"user", "system", "model"}, dropped.AuditRoles)

	// 仅含空串/空白时归一化后为空，必须回填默认 ["user"]（修复前存成空列表）。
	blankOnly, err := ParseStorageConfig(`{"audit_roles":[""]}`)
	require.NoError(t, err)
	require.Equal(t, []string{"user"}, blankOnly.AuditRoles)
	mixed, err := ParseStorageConfig(`{"audit_roles":["", "user", "USER", " model "]}`)
	require.NoError(t, err)
	require.Equal(t, []string{"user", "model"}, mixed.AuditRoles)

	// model（gemini 助手输出角色）可经请求路径配置并完整落库。
	modelReq := promptAuditUpdateRequest(1, 1, "")
	modelReq.AuditRoles = []string{"user", "model"}
	modelNext, err := manager.buildNextStorage(DefaultStorageConfig(), modelReq, 9)
	require.NoError(t, err)
	require.Equal(t, []string{"user", "model"}, modelNext.AuditRoles)

	// 请求路径拒绝未知角色，且不破坏现有合法 SaveConfig。
	badReq := promptAuditUpdateRequest(1, 1, "")
	badReq.AuditRoles = []string{"user", "evil"}
	err = validateUpdateConfigRequest(badReq)
	require.Error(t, err)
	require.Equal(t, "prompt_audit_invalid_audit_role", infraerrors.Reason(err))
	valid := promptAuditUpdateRequest(1, 1, "")
	require.NoError(t, validateUpdateConfigRequest(valid))
}

func TestUserGuardDefaultsOffForLegacyConfig(t *testing.T) {
	// 旧版本存储的配置没有 user_guard 段：解析后保持关闭且校验通过
	storage, err := ParseStorageConfig(`{"enabled":false,"strategy":"priority","worker_count":4,"queue_capacity":32768,"scanners":["pii"],"all_groups":true,"group_ids":[],"endpoints":[],"config_version":1}`)
	require.NoError(t, err)
	require.False(t, storage.UserGuard.Enabled)
	active, err := ActiveFromStorage(storage, true, prefixEncryptor{})
	require.NoError(t, err)
	require.False(t, active.UserGuard.Enabled)
}

func TestEndpointSystemPromptRoundTripAndValidation(t *testing.T) {
	// update → storage → active/public 全载体携带 system_prompt
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}
	request := promptAuditUpdateRequest(1, 1, "")
	request.Endpoints[0].SystemPrompt = "  " + DefaultGuardSystemPrompt + "  "
	next, err := manager.buildNextStorage(DefaultStorageConfig(), request, 9)
	require.NoError(t, err)
	// 存储时 trim
	require.Equal(t, DefaultGuardSystemPrompt, next.Endpoints[0].SystemPrompt)

	active, err := ActiveFromStorage(next, true, prefixEncryptor{})
	require.NoError(t, err)
	require.Equal(t, DefaultGuardSystemPrompt, active.Endpoints[0].SystemPrompt)

	public := PublicFromStorage(next, true, nil)
	require.Equal(t, DefaultGuardSystemPrompt, public.Endpoints[0].SystemPrompt)

	raw, err := json.Marshal(next)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"system_prompt"`)
	parsed, err := ParseStorageConfig(string(raw))
	require.NoError(t, err)
	require.Equal(t, DefaultGuardSystemPrompt, parsed.Endpoints[0].SystemPrompt)

	// 留空时 JSON 省略该字段（omitempty），保持旧端点 payload 形状不变
	request2 := promptAuditUpdateRequest(1, 1, "")
	next2, err := manager.buildNextStorage(DefaultStorageConfig(), request2, 9)
	require.NoError(t, err)
	raw2, err := json.Marshal(next2)
	require.NoError(t, err)
	require.NotContains(t, string(raw2), `"system_prompt"`)

	// 超长拒绝（storage 校验与 update 校验两条路径）
	tooLong := strings.Repeat("安", MaxSystemPromptChars+1)
	storage := DefaultStorageConfig()
	storage.Endpoints = []StorageEndpoint{{
		ID: "guard-1", Name: "Guard", Protocol: "openai_compatible", BaseURL: "http://127.0.0.1:8080",
		Model: DefaultGuardModel, TimeoutMS: 1000, InputLimit: 1000, Enabled: true, SystemPrompt: tooLong,
	}}
	err = validateStorageConfig(storage)
	require.Error(t, err)
	require.Equal(t, "prompt_audit_invalid_system_prompt", infraerrors.Reason(err))

	req3 := promptAuditUpdateRequest(1, 1, "")
	req3.Endpoints[0].SystemPrompt = tooLong
	err = validateUpdateConfigRequest(req3)
	require.Error(t, err)
	require.Equal(t, "prompt_audit_invalid_system_prompt", infraerrors.Reason(err))
}

func TestUserGuardWhitelistRoundTripAndValidation(t *testing.T) {
	// update → storage：去重 + 升序归一化，四种载体携带
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}
	request := promptAuditUpdateRequest(1, 1, "")
	request.UserGuard = UserGuardConfig{Enabled: true, Threshold: 3, WindowMinutes: 10, BanDurationMinutes: 60, WhitelistUserIDs: []int64{9, 3, 9, 5}}
	next, err := manager.buildNextStorage(DefaultStorageConfig(), request, 1)
	require.NoError(t, err)
	require.Equal(t, []int64{3, 5, 9}, next.UserGuard.WhitelistUserIDs)

	active, err := ActiveFromStorage(next, true, prefixEncryptor{})
	require.NoError(t, err)
	require.Equal(t, []int64{3, 5, 9}, active.UserGuard.WhitelistUserIDs)
	require.True(t, active.UserGuard.IsWhitelisted(5))
	require.False(t, active.UserGuard.IsWhitelisted(4))
	require.False(t, active.UserGuard.IsWhitelisted(0))

	public := PublicFromStorage(next, true, nil)
	require.Equal(t, []int64{3, 5, 9}, public.UserGuard.WhitelistUserIDs)

	raw, err := json.Marshal(next)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"whitelist_user_ids":[3,5,9]`)
	parsed, err := ParseStorageConfig(string(raw))
	require.NoError(t, err)
	require.Equal(t, []int64{3, 5, 9}, parsed.UserGuard.WhitelistUserIDs)

	// 非法值拒绝（update 原始路径与 storage 解析路径）
	req2 := promptAuditUpdateRequest(1, 1, "")
	req2.UserGuard = UserGuardConfig{WhitelistUserIDs: []int64{1, -2}}
	err = validateUpdateConfigRequest(req2)
	require.Error(t, err)
	require.Equal(t, "user_guard_invalid_whitelist_user", infraerrors.Reason(err))

	_, err = ParseStorageConfig(`{"user_guard":{"enabled":false,"whitelist_user_ids":[1,0]}}`)
	require.Error(t, err)
	require.Equal(t, "user_guard_invalid_whitelist_user", infraerrors.Reason(err))

	// 超上限拒绝
	tooMany := make([]int64, UserGuardMaxWhitelistSize+1)
	for i := range tooMany {
		tooMany[i] = int64(i + 1)
	}
	req3 := promptAuditUpdateRequest(1, 1, "")
	req3.UserGuard = UserGuardConfig{WhitelistUserIDs: tooMany}
	err = validateUpdateConfigRequest(req3)
	require.Error(t, err)
	require.Equal(t, "user_guard_invalid_whitelist", infraerrors.Reason(err))

	// 空列表/缺省保持合法（enabled 需配启用节点，这里只验证白名单字段本身）
	storage, err := ParseStorageConfig(`{"user_guard":{"enabled":false,"threshold":3,"window_minutes":10,"ban_duration_minutes":60}}`)
	require.NoError(t, err)
	require.Empty(t, storage.UserGuard.WhitelistUserIDs)
}

func TestDedupConfigDefaults(t *testing.T) {
	// 默认：查重开启，窗口 10 分钟；空配置与缺省字段同样回退默认。
	storage := DefaultStorageConfig()
	require.True(t, storage.DedupEnabled)
	require.Equal(t, DefaultDedupWindowMinutes, storage.DedupWindowMinutes)

	parsed, err := ParseStorageConfig("")
	require.NoError(t, err)
	require.True(t, parsed.DedupEnabled)
	require.Equal(t, DefaultDedupWindowMinutes, parsed.DedupWindowMinutes)

	legacy, err := ParseStorageConfig(`{"enabled":false,"config_version":9}`)
	require.NoError(t, err)
	require.True(t, legacy.DedupEnabled)
	require.Equal(t, DefaultDedupWindowMinutes, legacy.DedupWindowMinutes)

	active, err := ActiveFromStorage(legacy, true, prefixEncryptor{})
	require.NoError(t, err)
	require.True(t, active.DedupEnabled)
	require.Equal(t, DefaultDedupWindowMinutes, active.DedupWindowMinutes)

	public := PublicFromStorage(legacy, true, nil)
	require.True(t, public.DedupEnabled)
	require.Equal(t, DefaultDedupWindowMinutes, public.DedupWindowMinutes)
}

func TestDedupConfigRoundTrip(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}
	request := promptAuditUpdateRequest(1, 1, "")
	request.DedupEnabled = boolPtr(false)
	request.DedupWindowMinutes = intPtr(30)

	next, err := manager.buildNextStorage(DefaultStorageConfig(), request, 9)
	require.NoError(t, err)
	require.False(t, next.DedupEnabled)
	require.Equal(t, 30, next.DedupWindowMinutes)
	require.Contains(t, changeSummary(next), `"dedup_enabled":false`)
	require.Contains(t, changeSummary(next), `"dedup_window_minutes":30`)

	active, err := ActiveFromStorage(next, true, prefixEncryptor{})
	require.NoError(t, err)
	require.False(t, active.DedupEnabled)
	require.Equal(t, 30, active.DedupWindowMinutes)

	public := PublicFromStorage(next, true, nil)
	require.False(t, public.DedupEnabled)
	require.Equal(t, 30, public.DedupWindowMinutes)

	raw, err := json.Marshal(next)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"dedup_enabled":false`)
	require.Contains(t, string(raw), `"dedup_window_minutes":30`)
	parsed, err := ParseStorageConfig(string(raw))
	require.NoError(t, err)
	require.False(t, parsed.DedupEnabled)
	require.Equal(t, 30, parsed.DedupWindowMinutes)

	// 显式开启 + 自定义窗口也可完整往返
	request.DedupEnabled = boolPtr(true)
	request.DedupWindowMinutes = intPtr(5)
	on, err := manager.buildNextStorage(parsed, request, 9)
	require.NoError(t, err)
	require.True(t, on.DedupEnabled)
	require.Equal(t, 5, on.DedupWindowMinutes)
}

func TestDedupUpdateOmitsFieldPreservesStorage(t *testing.T) {
	manager := &ConfigManager{encryptor: prefixEncryptor{}, encryptionKeyConfigured: true}

	// update 省略 dedup_enabled/dedup_window_minutes（nil）：保留存储现值，
	// 防止 UI 保存把查重开关/窗口打回默认。
	for _, enabled := range []bool{true, false} {
		for _, window := range []int{5, 30} {
			base := DefaultStorageConfig()
			base.DedupEnabled = enabled
			base.DedupWindowMinutes = window
			request := promptAuditUpdateRequest(1, 1, "")
			next, err := manager.buildNextStorage(base, request, 9)
			require.NoError(t, err)
			require.Equal(t, enabled, next.DedupEnabled)
			require.Equal(t, window, next.DedupWindowMinutes)
		}
	}

	// 显式传值时正确覆盖存储值（双向）。
	for _, enabled := range []bool{true, false} {
		base := DefaultStorageConfig()
		base.DedupEnabled = !enabled
		request := promptAuditUpdateRequest(1, 1, "")
		request.DedupEnabled = boolPtr(enabled)
		next, err := manager.buildNextStorage(base, request, 9)
		require.NoError(t, err)
		require.Equal(t, enabled, next.DedupEnabled)
	}
	base := DefaultStorageConfig()
	base.DedupWindowMinutes = 60
	request := promptAuditUpdateRequest(1, 1, "")
	request.DedupWindowMinutes = intPtr(3)
	next, err := manager.buildNextStorage(base, request, 9)
	require.NoError(t, err)
	require.Equal(t, 3, next.DedupWindowMinutes)
}

func TestDedupWindowValidation(t *testing.T) {
	// storage 解析路径：窗口 0 由 normalize 回填默认；负值/超上限拒绝。
	zero, err := ParseStorageConfig(`{"dedup_window_minutes":0,"config_version":9}`)
	require.NoError(t, err)
	require.Equal(t, DefaultDedupWindowMinutes, zero.DedupWindowMinutes)

	for _, window := range []int{-1, MaxDedupWindowMinutes + 1} {
		cfg := DefaultStorageConfig()
		cfg.DedupWindowMinutes = window
		err := validateStorageConfig(cfg)
		require.Error(t, err)
		require.Equal(t, "prompt_audit_invalid_dedup_window", infraerrors.Reason(err))
	}

	// update 请求路径：显式传 0/负值/超上限拒绝（指针非 nil 才校验，nil 表示未提交）。
	for _, window := range []int{0, -1, MaxDedupWindowMinutes + 1} {
		req := promptAuditUpdateRequest(1, 1, "")
		req.DedupWindowMinutes = intPtr(window)
		err := validateUpdateConfigRequest(req)
		require.Error(t, err)
		require.Equal(t, "prompt_audit_invalid_dedup_window", infraerrors.Reason(err))
	}

	// nil 指针不校验（省略字段保留存储现值）。
	req := promptAuditUpdateRequest(1, 1, "")
	require.NoError(t, validateUpdateConfigRequest(req))// Regression coverage for issue #5732: refreshLoop reloads every 5s, so
// config_loaded must stay a change signal instead of ~17k identical lines a
// day, while still reporting the first load, real config changes and a
// recovery from a failed reload.
func TestConfigLoadedIsLoggedOnlyWhenSomethingChanged(t *testing.T) {
	storage := DefaultStorageConfig()
	storage.ConfigVersion = 4
	raw, err := json.Marshal(storage)
	require.NoError(t, err)
	repository := &switchableSettingRepository{staticSettingRepository: staticSettingRepository{values: map[string]string{
		SettingKeyPromptAuditConfig: string(raw),
		SettingKeyRiskControl:       "false",
	}}}
	manager := NewConfigManager(nil, repository, nil, prefixEncryptor{}, testTotpKeyConfig())

	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	loadedCount := func() int { return strings.Count(output.String(), EventConfigLoaded) }

	require.NoError(t, manager.Reload(context.Background()))
	require.Equal(t, 1, loadedCount(), "the first successful load must be logged")

	require.NoError(t, manager.Reload(context.Background()))
	require.NoError(t, manager.Reload(context.Background()))
	require.Equal(t, 1, loadedCount(), "TTL refreshes of an unchanged config must stay silent")

	repository.values[SettingKeyRiskControl] = "true"
	require.NoError(t, manager.Reload(context.Background()))
	require.Equal(t, 2, loadedCount(), "flipping the global risk control gate must be logged")

	storage.ConfigVersion = 5
	raw, err = json.Marshal(storage)
	require.NoError(t, err)
	repository.values[SettingKeyPromptAuditConfig] = string(raw)
	require.NoError(t, manager.Reload(context.Background()))
	require.Equal(t, 3, loadedCount(), "a new config version must be logged")

	repository.loadErr = errors.New("settings unavailable")
	require.Error(t, manager.Reload(context.Background()))
	require.Equal(t, 3, loadedCount(), "a failed reload must not claim a load")

	repository.loadErr = nil
	require.NoError(t, manager.Reload(context.Background()))
	require.Equal(t, 4, loadedCount(), "recovering from a failed reload must be visible")
}
