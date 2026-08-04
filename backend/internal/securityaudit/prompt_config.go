package securityaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	DefaultWorkerCount   = 4
	MaxWorkerCount       = 32
	DefaultQueueCapacity = 32768
	MaxQueueCapacity     = 100000
	DefaultTimeoutMS     = 3000
	MinTimeoutMS         = 100
	MaxTimeoutMS         = 30000
	DefaultInputLimit    = 4000
	MinInputLimit        = 128
	MaxInputLimit        = 100000
	// MaxSystemPromptChars 端点级系统提示词长度上限（Unicode 字符）。
	MaxSystemPromptChars = 8000
	DefaultPayloadTTL    = 30 * time.Minute
)

type SecretEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// ConfigStore is the injectable boundary between hot-path prompt auditing and
// the concrete settings/PostgreSQL/Redis-backed configuration manager.
type ConfigStore interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Active() (ActiveConfig, bool)
	EffectiveMode() Mode
	// BlockingActivationDegraded is true when storage intent requires blocking
	// but no usable blocking snapshot is active (cold start or failed reload).
	// It must stay false when blocking is not intended, even if config is
	// untrusted—otherwise default-off deployments fail closed for all traffic.
	BlockingActivationDegraded() bool
	Public() (PublicConfig, error)
	Save(ctx context.Context, req UpdateConfigRequest, actorID int64) (PublicConfig, error)
	RuntimeState() (expected int64, active int64, loadedAt *time.Time, loadError string)
	Encrypt(value string) (string, error)
	Decrypt(value string) (string, error)
}

type StorageEndpoint struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Protocol        string `json:"protocol"`
	BaseURL         string `json:"base_url"`
	Model           string `json:"model"`
	TokenCiphertext string `json:"token_ciphertext,omitempty"`
	TimeoutMS       int    `json:"timeout_ms"`
	InputLimit      int    `json:"input_limit"`
	// SystemPrompt 可选的端点级系统提示词：配置后审核请求以
	// [{role:"system"},{role:"user"}] 发送，使任意通用 chat completions
	// 模型（GPT/DeepSeek/Kimi 等）都能充当内容审核员；留空则保持裸内容
	// 单条 user 消息（适用于真正的 Qwen3Guard 官方端点）。非敏感，不加密。
	SystemPrompt string `json:"system_prompt,omitempty"`
	Enabled      bool   `json:"enabled"`
}

// UserGuardConfig 用户级内容违规守护配置：复用审计节点池在账号选定后、
// 发往上游前同步判定请求内容，窗口内违规次数达到阈值时临时封禁该用户。
type UserGuardConfig struct {
	Enabled            bool `json:"enabled"`
	Threshold          int  `json:"threshold"`
	WindowMinutes      int  `json:"window_minutes"`
	BanDurationMinutes int  `json:"ban_duration_minutes"`
	// WhitelistUserIDs 白名单用户 ID 列表：白名单内用户的请求完全跳过审核
	// （不调用审核模型、不计数、不封禁）。归一化后去重且升序。
	WhitelistUserIDs []int64 `json:"whitelist_user_ids"`
}

const (
	UserGuardMinThreshold       = 1
	UserGuardMaxThreshold       = 100
	UserGuardMinWindowMinutes   = 1
	UserGuardMaxWindowMinutes   = 1440
	UserGuardMinBanMinutes      = 1
	UserGuardMaxBanMinutes      = 10080
	UserGuardDefaultThreshold   = 3
	UserGuardDefaultWindowMin   = 10
	UserGuardDefaultBanDuration = 60
	// UserGuardMaxWhitelistSize 白名单用户 ID 数量上限。
	UserGuardMaxWhitelistSize = 500
)

// IsWhitelisted 判断用户是否在违规守护白名单内。
// WhitelistUserIDs 经 normalize 后保持升序，这里用二分查找。
func (cfg UserGuardConfig) IsWhitelisted(userID int64) bool {
	if userID <= 0 || len(cfg.WhitelistUserIDs) == 0 {
		return false
	}
	i := sort.Search(len(cfg.WhitelistUserIDs), func(i int) bool { return cfg.WhitelistUserIDs[i] >= userID })
	return i < len(cfg.WhitelistUserIDs) && cfg.WhitelistUserIDs[i] == userID
}

// storageConfig 的 AuditRoles 字段：提取审计文本时保留的消息角色；
// 缺省/为空时按默认 DefaultAuditRoles（仅 user）生效，详见 canonicalAuditRoles。
type storageConfig struct {
	Enabled                bool              `json:"enabled"`
	BlockingEnabled        bool              `json:"blocking_enabled"`
	BlockingLatestTurnOnly bool              `json:"blocking_latest_turn_only"`
	AsyncLatestTurnOnly    bool              `json:"async_latest_turn_only"`
	StorePassEvents        bool              `json:"store_pass_events"`
	Strategy               string            `json:"strategy"`
	WorkerCount            int               `json:"worker_count"`
	QueueCapacity          int               `json:"queue_capacity"`
	Scanners               []string          `json:"scanners"`
	AuditRoles             []string          `json:"audit_roles"`
	AllGroups              bool              `json:"all_groups"`
	GroupIDs               []int64           `json:"group_ids"`
	Endpoints              []StorageEndpoint `json:"endpoints"`
	UserGuard              UserGuardConfig   `json:"user_guard"`
	ConfigVersion          int64             `json:"config_version"`
	UpdatedAt              time.Time         `json:"updated_at"`
	UpdatedBy              int64             `json:"updated_by"`
	ChangeSummary          string            `json:"change_summary"`
}

type ActiveEndpoint struct {
	ID           string
	Name         string
	Protocol     string
	BaseURL      string
	Model        string
	Token        string
	TimeoutMS    int
	InputLimit   int
	SystemPrompt string
	Enabled      bool
	// TokenInvalid marks an endpoint whose persisted token ciphertext cannot be
	// decrypted with the current encryption key (key changed or auto-generated
	// on restart). The endpoint is kept visible for admins but excluded from
	// runtime use until the token is re-entered or cleared (issue #4887).
	TokenInvalid bool
}

type ActiveConfig struct {
	RiskControlEnabled     bool
	Enabled                bool
	BlockingEnabled        bool
	BlockingLatestTurnOnly bool
	AsyncLatestTurnOnly    bool
	StorePassEvents        bool
	Strategy               string
	WorkerCount            int
	QueueCapacity          int
	Scanners               []string
	AuditRoles             []string
	AllGroups              bool
	GroupIDs               []int64
	Endpoints              []ActiveEndpoint
	UserGuard              UserGuardConfig
	ConfigVersion          int64
	UpdatedAt              time.Time
	UpdatedBy              int64
	ChangeSummary          string
}

type PublicEndpoint struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	BaseURL      string `json:"base_url"`
	Model        string `json:"model"`
	TimeoutMS    int    `json:"timeout_ms"`
	InputLimit   int    `json:"input_limit"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	Enabled      bool   `json:"enabled"`
	HasToken     bool   `json:"has_token"`
	TokenStatus  string `json:"token_status"`
}

type PublicConfig struct {
	Enabled                bool             `json:"enabled"`
	BlockingEnabled        bool             `json:"blocking_enabled"`
	BlockingLatestTurnOnly bool             `json:"blocking_latest_turn_only"`
	AsyncLatestTurnOnly    bool             `json:"async_latest_turn_only"`
	StorePassEvents        bool             `json:"store_pass_events"`
	EffectiveMode          Mode             `json:"effective_mode"`
	Strategy               string           `json:"strategy"`
	WorkerCount            int              `json:"worker_count"`
	QueueCapacity          int              `json:"queue_capacity"`
	Scanners               []string         `json:"scanners"`
	AuditRoles             []string         `json:"audit_roles"`
	AllGroups              bool             `json:"all_groups"`
	GroupIDs               []int64          `json:"group_ids"`
	Endpoints              []PublicEndpoint `json:"endpoints"`
	UserGuard              UserGuardConfig  `json:"user_guard"`
	ConfigVersion          int64            `json:"config_version"`
	UpdatedAt              time.Time        `json:"updated_at"`
	UpdatedBy              int64            `json:"updated_by"`
	ChangeSummary          string           `json:"change_summary"`
}

type UpdateEndpoint struct {
	ID           string `json:"id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Protocol     string `json:"protocol"`
	BaseURL      string `json:"base_url" binding:"required"`
	Model        string `json:"model"`
	Token        string `json:"token,omitempty"`
	ClearToken   bool   `json:"clear_token"`
	TimeoutMS    int    `json:"timeout_ms"`
	InputLimit   int    `json:"input_limit"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	Enabled      bool   `json:"enabled"`
}

type UpdateConfigRequest struct {
	ExpectedConfigVersion  int64 `json:"expected_config_version" binding:"required"`
	Enabled                bool  `json:"enabled"`
	BlockingEnabled        bool  `json:"blocking_enabled"`
	BlockingLatestTurnOnly bool  `json:"blocking_latest_turn_only"`
	// AsyncLatestTurnOnly 为指针以区分「未提交」与「显式 false」：前端 UI 保存
	// 未携带该字段时（nil）保留存储现值，避免配置被静默重置为 false。
	AsyncLatestTurnOnly *bool            `json:"async_latest_turn_only"`
	StorePassEvents     bool             `json:"store_pass_events"`
	Strategy            string           `json:"strategy"`
	WorkerCount         int              `json:"worker_count"`
	QueueCapacity       int              `json:"queue_capacity"`
	Scanners            []string         `json:"scanners"`
	AuditRoles          []string         `json:"audit_roles"`
	AllGroups           bool             `json:"all_groups"`
	GroupIDs            []int64          `json:"group_ids"`
	Endpoints           []UpdateEndpoint `json:"endpoints"`
	UserGuard           UserGuardConfig  `json:"user_guard"`
}

// DefaultAuditRoles 默认只提取 user 角色消息用于审计。系统提示词、开发者指令、
// assistant/tool 轮次主要由平台方或上游工具控制，且体量巨大（如 Codex 巨型系统
// 提示词，生产日志 input_chars 曾高达 146952），全量提取会打爆审核模型的输入上限；
// 管理员可按分组在配置中显式加入 system/developer/assistant/model/tool。
var DefaultAuditRoles = []string{"user"}

// AuditRoleNames 管理员可选的审计提取角色。model 是 gemini 协议的助手输出角色
// （OpenAI 侧 assistant 的对应物），isAssistantOutputSegment 同时识别两者，
// 因此配置层一并开放，避免管理员配了 assistant 也恢复不了 gemini 的 model 轮次。
var AuditRoleNames = []string{"user", "system", "developer", "assistant", "model", "tool"}

func isAuditRoleName(role string) bool {
	normalized := strings.ToLower(strings.TrimSpace(role))
	for _, name := range AuditRoleNames {
		if name == normalized {
			return true
		}
	}
	return false
}

// canonicalAuditRoles 归一化审计角色：小写、trim、去重、丢弃未知角色，按
// AuditRoleNames 顺序输出。存储配置中的未知角色静默丢弃（与
// canonicalScannerIDs 一致），避免旧库或手改配置拖垮整个配置加载。
func canonicalAuditRoles(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		role := strings.ToLower(strings.TrimSpace(value))
		if isAuditRoleName(role) {
			seen[role] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for _, role := range AuditRoleNames {
		if _, ok := seen[role]; ok {
			result = append(result, role)
		}
	}
	return result
}

// validateAuditRoles 校验审计提取角色名合法性；空列表合法（由 normalize 填默认值）。
func validateAuditRoles(roles []string) error {
	for _, role := range roles {
		if !isAuditRoleName(role) {
			return infraerrors.BadRequest("prompt_audit_invalid_audit_role", "提示词审计提取角色仅支持 user/system/developer/assistant/model/tool")
		}
	}
	return nil
}

func DefaultStorageConfig() storageConfig {
	return storageConfig{
		Enabled:                false,
		BlockingEnabled:        false,
		BlockingLatestTurnOnly: false,
		AsyncLatestTurnOnly:    true,
		StorePassEvents:        false,
		Strategy:               "priority",
		WorkerCount:            DefaultWorkerCount,
		QueueCapacity:          DefaultQueueCapacity,
		Scanners:               append([]string(nil), AllScannerIDs...),
		AuditRoles:             append([]string(nil), DefaultAuditRoles...),
		AllGroups:              true,
		GroupIDs:               []int64{},
		Endpoints:              []StorageEndpoint{},
		ConfigVersion:          1,
	}
}

func ParseStorageConfig(raw string) (storageConfig, error) {
	cfg := DefaultStorageConfig()
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return storageConfig{}, fmt.Errorf("decode prompt audit config: %w", err)
	}
	normalizeStorageConfig(&cfg)
	if err := validateStorageConfig(cfg); err != nil {
		return storageConfig{}, err
	}
	return cfg, nil
}

func normalizeStorageConfig(cfg *storageConfig) {
	if cfg == nil {
		return
	}
	if cfg.ConfigVersion < 1 {
		cfg.ConfigVersion = 1
	}
	if strings.TrimSpace(cfg.Strategy) == "" {
		cfg.Strategy = "priority"
	}
	if cfg.WorkerCount == 0 {
		cfg.WorkerCount = DefaultWorkerCount
	}
	if cfg.QueueCapacity == 0 {
		cfg.QueueCapacity = DefaultQueueCapacity
	}
	if len(cfg.Scanners) == 0 {
		cfg.Scanners = append([]string(nil), AllScannerIDs...)
	}
	cfg.Scanners = canonicalScannerIDs(cfg.Scanners)
	// 先归一化再判空回填：[""]/[" "] 归一化后为空列表也必须回填默认，
	// 保持「审计角色永不为空 → 默认 ["user"]」不变量。
	cfg.AuditRoles = canonicalAuditRoles(cfg.AuditRoles)
	if len(cfg.AuditRoles) == 0 {
		cfg.AuditRoles = append([]string(nil), DefaultAuditRoles...)
	}
	cfg.GroupIDs = canonicalInt64s(cfg.GroupIDs)
	// 白名单只去重+排序，保留非法值（<=0）交由 validate 拒绝，
	// 避免静默改变管理员的配置意图。
	cfg.UserGuard.WhitelistUserIDs = dedupeSortedInt64s(cfg.UserGuard.WhitelistUserIDs)
	// Preserve an invalid blocking-without-audit combination so validation can
	// reject it instead of silently changing administrator intent.
	for i := range cfg.Endpoints {
		ep := &cfg.Endpoints[i]
		ep.ID = strings.TrimSpace(ep.ID)
		ep.Name = strings.TrimSpace(ep.Name)
		ep.Protocol = strings.TrimSpace(ep.Protocol)
		if ep.Protocol == "" {
			ep.Protocol = "openai_compatible"
		}
		ep.BaseURL = strings.TrimSpace(ep.BaseURL)
		ep.Model = strings.TrimSpace(ep.Model)
		if ep.Model == "" {
			ep.Model = DefaultGuardModel
		}
		if ep.TimeoutMS == 0 {
			ep.TimeoutMS = DefaultTimeoutMS
		}
		if ep.InputLimit == 0 {
			ep.InputLimit = DefaultInputLimit
		}
	}
}

func validateStorageConfig(cfg storageConfig) error {
	if cfg.BlockingEnabled && !cfg.Enabled {
		return infraerrors.BadRequest(ErrorCodeRequiresEnabled, "开启同步阻止前必须先启用提示词审计")
	}
	if cfg.Strategy != "priority" {
		return infraerrors.BadRequest("prompt_audit_invalid_strategy", "提示词审计策略仅支持 priority")
	}
	if cfg.WorkerCount < 1 || cfg.WorkerCount > MaxWorkerCount {
		return infraerrors.BadRequest("prompt_audit_invalid_worker_count", "Worker 数量超出允许范围")
	}
	if cfg.QueueCapacity < 1 || cfg.QueueCapacity > MaxQueueCapacity {
		return infraerrors.BadRequest("prompt_audit_invalid_queue_capacity", "队列容量超出允许范围")
	}
	if !cfg.AllGroups && len(cfg.GroupIDs) == 0 {
		return infraerrors.BadRequest("prompt_audit_groups_required", "指定分组模式至少需要选择一个分组")
	}
	if len(cfg.Scanners) == 0 {
		return infraerrors.BadRequest("prompt_audit_scanners_required", "至少需要启用一个风险分类")
	}
	if err := validateAuditRoles(cfg.AuditRoles); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(cfg.Endpoints))
	enabled := 0
	for _, ep := range cfg.Endpoints {
		if ep.ID == "" || ep.Name == "" {
			return infraerrors.BadRequest("prompt_audit_invalid_endpoint", "审计节点 ID 和名称不能为空")
		}
		if _, ok := seen[ep.ID]; ok {
			return infraerrors.BadRequest("prompt_audit_duplicate_endpoint", "审计节点 ID 不能重复")
		}
		seen[ep.ID] = struct{}{}
		if ep.Protocol != "openai_compatible" {
			return infraerrors.BadRequest("prompt_audit_invalid_endpoint_protocol", "审计节点仅支持 OpenAI 兼容协议")
		}
		if _, err := NormalizeBaseURL(ep.BaseURL); err != nil {
			return err
		}
		if ep.TimeoutMS < MinTimeoutMS || ep.TimeoutMS > MaxTimeoutMS {
			return infraerrors.BadRequest("prompt_audit_invalid_timeout", "审计节点超时超出允许范围")
		}
		if ep.InputLimit < MinInputLimit || ep.InputLimit > MaxInputLimit {
			return infraerrors.BadRequest("prompt_audit_invalid_input_limit", "审计节点输入上限超出允许范围")
		}
		if len([]rune(ep.SystemPrompt)) > MaxSystemPromptChars {
			return infraerrors.BadRequest("prompt_audit_invalid_system_prompt", "审计节点系统提示词超出长度上限（8000 字符）")
		}
		if ep.Enabled {
			enabled++
		}
	}
	if cfg.Enabled && enabled == 0 {
		return infraerrors.BadRequest("prompt_audit_endpoint_required", "启用提示词审计前至少需要启用一个审计节点")
	}
	if err := validateUserGuardConfig(cfg.UserGuard); err != nil {
		return err
	}
	if cfg.UserGuard.Enabled && enabled == 0 {
		return infraerrors.BadRequest("prompt_audit_endpoint_required", "启用用户违规守护前至少需要启用一个审计节点")
	}
	return nil
}

func validateUserGuardConfig(guard UserGuardConfig) error {
	if len(guard.WhitelistUserIDs) > UserGuardMaxWhitelistSize {
		return infraerrors.BadRequest("user_guard_invalid_whitelist", "用户违规守护白名单超出数量上限（500 个）")
	}
	for _, id := range guard.WhitelistUserIDs {
		if id <= 0 {
			return infraerrors.BadRequest("user_guard_invalid_whitelist_user", "用户违规守护白名单包含无效用户 ID（必须为正整数）")
		}
	}
	if !guard.Enabled {
		return nil
	}
	if guard.Threshold < UserGuardMinThreshold || guard.Threshold > UserGuardMaxThreshold {
		return infraerrors.BadRequest("user_guard_invalid_threshold", "用户违规守护阈值超出允许范围（1-100）")
	}
	if guard.WindowMinutes < UserGuardMinWindowMinutes || guard.WindowMinutes > UserGuardMaxWindowMinutes {
		return infraerrors.BadRequest("user_guard_invalid_window", "用户违规守护统计窗口超出允许范围（1-1440 分钟）")
	}
	if guard.BanDurationMinutes < UserGuardMinBanMinutes || guard.BanDurationMinutes > UserGuardMaxBanMinutes {
		return infraerrors.BadRequest("user_guard_invalid_ban_duration", "用户违规守护封禁时长超出允许范围（1-10080 分钟）")
	}
	return nil
}

func validateUpdateConfigRequest(req UpdateConfigRequest) error {
	if strings.TrimSpace(req.Strategy) != "priority" {
		return infraerrors.BadRequest("prompt_audit_invalid_strategy", "提示词审计策略仅支持 priority")
	}
	if req.WorkerCount < 1 || req.WorkerCount > MaxWorkerCount {
		return infraerrors.BadRequest("prompt_audit_invalid_worker_count", "Worker 数量超出允许范围")
	}
	if req.QueueCapacity < 1 || req.QueueCapacity > MaxQueueCapacity {
		return infraerrors.BadRequest("prompt_audit_invalid_queue_capacity", "队列容量超出允许范围")
	}
	if len(req.Scanners) == 0 {
		return infraerrors.BadRequest("prompt_audit_scanners_required", "至少需要启用一个风险分类")
	}
	for _, scanner := range req.Scanners {
		if _, ok := ScannerCatalog[NormalizeCategory(scanner)]; !ok {
			return infraerrors.BadRequest("prompt_audit_invalid_scanner", "提示词审计风险分类无效")
		}
	}
	if err := validateAuditRoles(req.AuditRoles); err != nil {
		return err
	}
	if !req.AllGroups {
		if len(req.GroupIDs) == 0 {
			return infraerrors.BadRequest("prompt_audit_groups_required", "指定分组模式至少需要选择一个分组")
		}
		for _, groupID := range req.GroupIDs {
			if groupID <= 0 {
				return infraerrors.BadRequest("prompt_audit_invalid_group", "提示词审计分组 ID 无效")
			}
		}
	}
	for _, endpoint := range req.Endpoints {
		if endpoint.TimeoutMS < MinTimeoutMS || endpoint.TimeoutMS > MaxTimeoutMS {
			return infraerrors.BadRequest("prompt_audit_invalid_timeout", "审计节点超时超出允许范围")
		}
		if endpoint.InputLimit < MinInputLimit || endpoint.InputLimit > MaxInputLimit {
			return infraerrors.BadRequest("prompt_audit_invalid_input_limit", "审计节点输入上限超出允许范围")
		}
		if len([]rune(endpoint.SystemPrompt)) > MaxSystemPromptChars {
			return infraerrors.BadRequest("prompt_audit_invalid_system_prompt", "审计节点系统提示词超出长度上限（8000 字符）")
		}
	}
	if err := validateUserGuardConfig(req.UserGuard); err != nil {
		return err
	}
	return nil
}

func (cfg ActiveConfig) EffectiveMode() Mode {
	if !cfg.RiskControlEnabled || !cfg.Enabled {
		return ModeOff
	}
	if cfg.BlockingEnabled {
		return ModeBlocking
	}
	return ModeAsync
}

func (cfg ActiveConfig) IncludesGroup(groupID *int64) bool {
	if cfg.AllGroups {
		return true
	}
	if groupID == nil {
		return false
	}
	i := sort.Search(len(cfg.GroupIDs), func(i int) bool { return cfg.GroupIDs[i] >= *groupID })
	return i < len(cfg.GroupIDs) && cfg.GroupIDs[i] == *groupID
}

func (cfg ActiveConfig) EnabledEndpoints() []ActiveEndpoint {
	result := make([]ActiveEndpoint, 0, len(cfg.Endpoints))
	for _, ep := range cfg.Endpoints {
		if ep.Enabled {
			result = append(result, ep)
		}
	}
	return result
}

// InvalidTokenEndpointIDs lists endpoints whose stored token could not be
// decrypted with the current encryption key.
func (cfg ActiveConfig) InvalidTokenEndpointIDs() []string {
	ids := make([]string, 0)
	for _, ep := range cfg.Endpoints {
		if ep.TokenInvalid {
			ids = append(ids, ep.ID)
		}
	}
	return ids
}

func PublicFromStorage(cfg storageConfig, riskControlEnabled bool, invalidTokenEndpointIDs []string) PublicConfig {
	invalid := make(map[string]struct{}, len(invalidTokenEndpointIDs))
	for _, id := range invalidTokenEndpointIDs {
		invalid[id] = struct{}{}
	}
	scanners := append([]string{}, cfg.Scanners...)
	groupIDs := append([]int64{}, cfg.GroupIDs...)
	endpoints := make([]PublicEndpoint, 0, len(cfg.Endpoints))
	for _, ep := range cfg.Endpoints {
		hasToken := strings.TrimSpace(ep.TokenCiphertext) != ""
		status := "missing"
		if hasToken {
			status = "configured"
			if _, ok := invalid[ep.ID]; ok {
				status = "invalid"
			}
		}
		endpoints = append(endpoints, PublicEndpoint{
			ID: ep.ID, Name: ep.Name, Protocol: ep.Protocol, BaseURL: ep.BaseURL,
			Model: ep.Model, TimeoutMS: ep.TimeoutMS, InputLimit: ep.InputLimit,
			SystemPrompt: ep.SystemPrompt,
			Enabled:      ep.Enabled, HasToken: hasToken, TokenStatus: status,
		})
	}
	active := ActiveConfig{RiskControlEnabled: riskControlEnabled, Enabled: cfg.Enabled, BlockingEnabled: cfg.BlockingEnabled}
	return PublicConfig{
		Enabled: cfg.Enabled, BlockingEnabled: cfg.BlockingEnabled, BlockingLatestTurnOnly: cfg.BlockingLatestTurnOnly, AsyncLatestTurnOnly: cfg.AsyncLatestTurnOnly, StorePassEvents: cfg.StorePassEvents,
		EffectiveMode: active.EffectiveMode(), Strategy: cfg.Strategy, WorkerCount: cfg.WorkerCount,
		QueueCapacity: cfg.QueueCapacity, Scanners: scanners, AuditRoles: append([]string(nil), cfg.AuditRoles...), AllGroups: cfg.AllGroups,
		GroupIDs: groupIDs, Endpoints: endpoints, UserGuard: cfg.UserGuard, ConfigVersion: cfg.ConfigVersion,
		UpdatedAt: cfg.UpdatedAt, UpdatedBy: cfg.UpdatedBy, ChangeSummary: cfg.ChangeSummary,
	}
}

func ActiveFromStorage(cfg storageConfig, riskControlEnabled bool, encryptor SecretEncryptor) (ActiveConfig, error) {
	active := ActiveConfig{
		RiskControlEnabled: riskControlEnabled, Enabled: cfg.Enabled, BlockingEnabled: cfg.BlockingEnabled,
		BlockingLatestTurnOnly: cfg.BlockingLatestTurnOnly,
		AsyncLatestTurnOnly:    cfg.AsyncLatestTurnOnly,
		StorePassEvents:        cfg.StorePassEvents, Strategy: cfg.Strategy, WorkerCount: cfg.WorkerCount,
		QueueCapacity: cfg.QueueCapacity, Scanners: append([]string(nil), cfg.Scanners...),
		AuditRoles: append([]string(nil), cfg.AuditRoles...), AllGroups: cfg.AllGroups,
		GroupIDs: append([]int64(nil), cfg.GroupIDs...), UserGuard: cfg.UserGuard, ConfigVersion: cfg.ConfigVersion,
		UpdatedAt: cfg.UpdatedAt, UpdatedBy: cfg.UpdatedBy, ChangeSummary: cfg.ChangeSummary,
		Endpoints: make([]ActiveEndpoint, 0, len(cfg.Endpoints)),
	}
	for _, ep := range cfg.Endpoints {
		token := ""
		tokenInvalid := false
		if ep.TokenCiphertext != "" {
			if encryptor == nil {
				return ActiveConfig{}, fmt.Errorf("prompt audit secret encryptor unavailable")
			}
			plain, err := encryptor.Decrypt(ep.TokenCiphertext)
			if err != nil {
				// An undecryptable token (encryption key changed or regenerated)
				// must not take the whole config down: admins would otherwise be
				// locked out of the real config version and unable to recover
				// (issue #4887). Keep the ciphertext persisted, but exclude the
				// endpoint from runtime use until the token is re-entered.
				tokenInvalid = true
			} else {
				token = plain
			}
		}
		active.Endpoints = append(active.Endpoints, ActiveEndpoint{
			ID: ep.ID, Name: ep.Name, Protocol: ep.Protocol, BaseURL: ep.BaseURL, Model: ep.Model,
			Token: token, TimeoutMS: ep.TimeoutMS, InputLimit: ep.InputLimit,
			SystemPrompt: ep.SystemPrompt,
			Enabled:      ep.Enabled && !tokenInvalid, TokenInvalid: tokenInvalid,
		})
	}
	return active, nil
}

func changeSummary(cfg storageConfig) string {
	summary := struct {
		Enabled                bool   `json:"enabled"`
		BlockingEnabled        bool   `json:"blocking_enabled"`
		BlockingLatestTurnOnly bool   `json:"blocking_latest_turn_only"`
		AsyncLatestTurnOnly    bool   `json:"async_latest_turn_only"`
		StorePassEvents        bool   `json:"store_pass_events"`
		EndpointCount          int    `json:"endpoint_count"`
		ScannerCount           int    `json:"scanner_count"`
		AllGroups              bool   `json:"all_groups"`
		GroupCount             int    `json:"group_count"`
		GroupHash              string `json:"group_hash"`
	}{cfg.Enabled, cfg.BlockingEnabled, cfg.BlockingLatestTurnOnly, cfg.AsyncLatestTurnOnly, cfg.StorePassEvents, len(cfg.Endpoints), len(cfg.Scanners), cfg.AllGroups, len(cfg.GroupIDs), ""}
	rawGroups, _ := json.Marshal(cfg.GroupIDs)
	digest := sha256.Sum256(rawGroups)
	summary.GroupHash = hex.EncodeToString(digest[:])
	raw, _ := json.Marshal(summary)
	return string(raw)
}

// dedupeSortedInt64s 去重并升序排序，但不丢弃任何值（含非法值）。
func dedupeSortedInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return []int64{}
	}
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func canonicalInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func canonicalScannerIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		id := NormalizeCategory(value)
		if _, ok := ScannerCatalog[id]; ok {
			seen[id] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for _, id := range AllScannerIDs {
		if _, ok := seen[id]; ok {
			result = append(result, id)
		}
	}
	return result
}
