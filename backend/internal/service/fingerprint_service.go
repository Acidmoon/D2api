package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// 模型指纹检测：异步任务编排。
// 参照 image_task 的「提交 → 任务 ID → 轮询」模式的简化版：任务状态只存内存
// （map + mutex），持久层是报告/参考 JSON 文件（fingerprint_store.go）。
// 探测执行复用监控模块的 SSRF 安全客户端（postRawJSON → safeDialContext）。

// 业务错误（统一在此声明，避免散落）。
var (
	ErrFingerprintInvalidTargetType = infraerrors.BadRequest(
		"FINGERPRINT_INVALID_TARGET_TYPE", "target_type must be account or external",
	)
	ErrFingerprintInvalidProvider = infraerrors.BadRequest(
		"FINGERPRINT_INVALID_PROVIDER", "provider must be one of openai/anthropic/gemini/grok",
	)
	ErrFingerprintMissingModel = infraerrors.BadRequest(
		"FINGERPRINT_MISSING_MODEL", "model and reference_model are required",
	)
	ErrFingerprintMissingAccount = infraerrors.BadRequest(
		"FINGERPRINT_MISSING_ACCOUNT", "account_id is required when target_type is account",
	)
	ErrFingerprintMissingExternal = infraerrors.BadRequest(
		"FINGERPRINT_MISSING_EXTERNAL", "base_url, api_key and provider are required when target_type is external",
	)
	ErrFingerprintInvalidEndpoint = infraerrors.BadRequest(
		"FINGERPRINT_INVALID_ENDPOINT", "base_url must be a valid https URL",
	)
	ErrFingerprintEndpointPrivate = infraerrors.BadRequest(
		"FINGERPRINT_ENDPOINT_PRIVATE", "base_url must be a public host",
	)
	ErrFingerprintEndpointUnreachable = infraerrors.BadRequest(
		"FINGERPRINT_ENDPOINT_UNREACHABLE", "base_url hostname could not be resolved",
	)
	ErrFingerprintUnsupportedPlatform = infraerrors.BadRequest(
		"FINGERPRINT_UNSUPPORTED_PLATFORM", "account platform is not supported for fingerprint audit (supported: anthropic/openai/gemini/grok)",
	)
	ErrFingerprintMissingCredential = infraerrors.BadRequest(
		"FINGERPRINT_MISSING_CREDENTIAL", "account has no usable api_key/access_token credential",
	)
	ErrFingerprintReferenceNotFound = infraerrors.BadRequest(
		"FINGERPRINT_REFERENCE_NOT_FOUND", "reference fingerprint not found for the model; register one first via POST /admin/fingerprint/references or pass reference_account_id",
	)
	ErrFingerprintReferenceEmpty = infraerrors.BadRequest(
		"FINGERPRINT_REFERENCE_EMPTY", "reference fingerprint is empty (0 valid samples in all cells); its registration likely failed, please re-register it",
	)
	ErrFingerprintAuditNotFound = infraerrors.NotFound(
		"FINGERPRINT_AUDIT_NOT_FOUND", "fingerprint audit not found",
	)
	ErrFingerprintAuditRunning = infraerrors.New(
		http.StatusConflict, "FINGERPRINT_AUDIT_RUNNING", "audit task is still running and cannot be deleted",
	)
)

// 报告 flags（设计文档 §8）：出现即从指纹证据中剔除对应 cell。
const (
	FingerprintFlagResponseCaching = "response_caching"
	FingerprintFlagHiddenReasoning = "hidden_reasoning"
	FingerprintFlagNotApplicable   = "not_applicable"
)

// cell 不进 JSD 的原因（报告 cell.excluded 字段）。
const (
	fingerprintExcludedResponseCaching     = "response_caching"
	fingerprintExcludedHiddenReasoning     = "hidden_reasoning"
	fingerprintExcludedInsufficientSamples = "insufficient_samples"
)

// 异步任务类型。
const (
	FingerprintTaskKindAudit             = "audit"
	FingerprintTaskKindRegisterReference = "register_reference"
)

// fingerprintBatteryTimeout 单轮电池（注册或检测各算一轮）的总超时。
const fingerprintBatteryTimeout = 30 * time.Minute

// geminiDefaultBaseURL Gemini 账号未配置 base_url 时的官方端点。
const geminiDefaultBaseURL = "https://generativelanguage.googleapis.com"

// anthropicDefaultBaseURL Anthropic OAuth 账号的官方端点（apikey 账号由 GetBaseURL 自带默认值）。
const anthropicDefaultBaseURL = "https://api.anthropic.com"

// FingerprintTaskStatus 内存中的异步任务状态（GET /audits/:id 进行中时返回）。
type FingerprintTaskStatus struct {
	TaskID         string                    `json:"task_id"`
	Kind           string                    `json:"kind"` // audit / register_reference
	Status         string                    `json:"status"`
	Progress       FingerprintReportProgress `json:"progress"`
	Model          string                    `json:"model"`
	ReferenceModel string                    `json:"reference_model,omitempty"`
	Error          string                    `json:"error,omitempty"`
	CreatedAt      time.Time                 `json:"created_at"`
	FinishedAt     *time.Time                `json:"finished_at,omitempty"`
}

// fingerprintTask 内存任务：互斥锁保护的状态快照；target 是检测目标摘要（列表行用）。
type fingerprintTask struct {
	mu     sync.Mutex
	status FingerprintTaskStatus
	target FingerprintReportTarget
}

func (t *fingerprintTask) snapshot() FingerprintTaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status
}

// incProgress 完成一个探测（成败都计数，失败样本不进数据）。
func (t *fingerprintTask) incProgress() {
	t.mu.Lock()
	t.status.Progress.Done++
	t.mu.Unlock()
}

func (t *fingerprintTask) finish(status string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status.Status = status
	if err != nil {
		t.status.Error = sanitizeErrorMessage(truncateMessage(err.Error()))
	}
	now := time.Now()
	t.status.FinishedAt = &now
}

// FingerprintAuditParams 是 StartAudit 的入参（handler 已做基础 binding 校验）。
type FingerprintAuditParams struct {
	TargetType         string // account / external
	AccountID          int64
	BaseURL            string
	APIKey             string
	Provider           string
	APIMode            string // openai 可选 chat_completions / responses
	Model              string
	ReferenceModel     string
	ReferenceAccountID int64 // >0 时先对该账号现场采样注册参考，再测目标
	KeepRaw            bool  // true 时报告附加原始回答样本
	Concurrency        int   // 并发 worker 数：0=默认 2，clamp [1,16]
	IntervalMs         *int  // 请求间隔毫秒：nil=默认 500，clamp [0,60000]
	OperatorID         int64
}

// FingerprintService 指纹检测服务：任务编排 + 文件存储。
type FingerprintService struct {
	accountSvc *AccountService
	store      *fingerprintStore

	mu    sync.Mutex
	tasks map[string]*fingerprintTask
}

// NewFingerprintService 创建指纹检测服务。数据目录默认 ./data/fingerprint。
func NewFingerprintService(accountSvc *AccountService, cfg *config.Config) *FingerprintService {
	dataDir := "./data/fingerprint"
	if cfg != nil && strings.TrimSpace(cfg.Fingerprint.DataDir) != "" {
		dataDir = strings.TrimSpace(cfg.Fingerprint.DataDir)
	}
	return &FingerprintService{
		accountSvc: accountSvc,
		store:      newFingerprintStore(dataDir),
		tasks:      make(map[string]*fingerprintTask),
	}
}

// newTask 登记一个 running 状态的内存任务。
func (s *FingerprintService) newTask(kind, model, referenceModel string, total int, target FingerprintReportTarget) *fingerprintTask {
	t := &fingerprintTask{
		status: FingerprintTaskStatus{
			TaskID:         uuid.NewString(),
			Kind:           kind,
			Status:         FingerprintStatusRunning,
			Progress:       FingerprintReportProgress{Done: 0, Total: total},
			Model:          model,
			ReferenceModel: referenceModel,
			CreatedAt:      time.Now(),
		},
		target: target,
	}
	s.mu.Lock()
	s.tasks[t.status.TaskID] = t
	s.mu.Unlock()
	return t
}

func (s *FingerprintService) getTask(id string) (*fingerprintTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	return t, ok
}

// ---------- 对外方法 ----------

// StartAudit 发起一次指纹检测：校验参数、解析目标，然后后台 goroutine 执行电池。
// 返回任务状态（含 task_id），前端轮询 GET /admin/fingerprint/audits/:id。
func (s *FingerprintService) StartAudit(ctx context.Context, params FingerprintAuditParams) (*FingerprintTaskStatus, error) {
	params.Model = strings.TrimSpace(params.Model)
	params.ReferenceModel = strings.TrimSpace(params.ReferenceModel)
	if params.Model == "" || params.ReferenceModel == "" {
		return nil, ErrFingerprintMissingModel
	}

	// 解析被测目标（凭证仅运行期持有，不落盘）。
	target, reportTarget, err := s.resolveTarget(ctx, params)
	if err != nil {
		return nil, err
	}

	// 参考基准：reference_account_id 提供时先现场采样注册，否则复用已注册文件。
	var refTarget *fingerprintProbeTarget
	var refAccountID *int64
	var reference *FingerprintReference
	if params.ReferenceAccountID > 0 {
		refTarget, _, err = s.resolveAccountTarget(ctx, params.ReferenceAccountID, params.ReferenceModel)
		if err != nil {
			return nil, err
		}
		id := params.ReferenceAccountID
		refAccountID = &id
	} else {
		reference, err = s.store.loadReference(params.ReferenceModel)
		if err != nil {
			return nil, err
		}
		// 空参考（所有 cell valid=0，通常是上次注册失败留下的坏文件）直接拒绝，
		// 避免跑完电池才给个莫名其妙的「证据不足」。
		if fingerprintReferenceEmpty(reference) {
			return nil, ErrFingerprintReferenceEmpty
		}
	}

	probesPerBattery := len(fingerprintCells()) * (fingerprintSamplesPerCell + fingerprintGreedySamplesPerCell)
	total := probesPerBattery
	if refTarget != nil {
		total += probesPerBattery
	}
	exec := fingerprintExecConfig{
		Concurrency: clampFingerprintConcurrency(params.Concurrency),
		IntervalMs:  clampFingerprintIntervalMs(params.IntervalMs),
	}
	task := s.newTask(FingerprintTaskKindAudit, params.Model, params.ReferenceModel, total, reportTarget)
	go s.executeAudit(task, target, reportTarget, refTarget, refAccountID, reference, params, exec)
	snap := task.snapshot()
	return &snap, nil
}

// StartReferenceRegistration 注册参考指纹：对可信账号现场采样，写 references/<model>.json。
// concurrency/intervalMs 语义同 StartAudit（0/nil 用默认值）。
func (s *FingerprintService) StartReferenceRegistration(ctx context.Context, accountID int64, model string, concurrency int, intervalMs *int) (*FingerprintTaskStatus, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, ErrFingerprintMissingModel
	}
	target, reportTarget, err := s.resolveAccountTarget(ctx, accountID, model)
	if err != nil {
		return nil, err
	}
	exec := fingerprintExecConfig{
		Concurrency: clampFingerprintConcurrency(concurrency),
		IntervalMs:  clampFingerprintIntervalMs(intervalMs),
	}
	total := len(fingerprintCells()) * (fingerprintSamplesPerCell + fingerprintGreedySamplesPerCell)
	task := s.newTask(FingerprintTaskKindRegisterReference, model, "", total, reportTarget)
	go s.executeReferenceRegistration(task, target, accountID, model, exec)
	snap := task.snapshot()
	return &snap, nil
}

// GetAudit 查询检测详情/进度：进行中返回内存任务状态（report 为 nil），
// 已完成/失败返回报告文件内容（report 非 nil）。
func (s *FingerprintService) GetAudit(id string) (*FingerprintTaskStatus, *FingerprintReport, error) {
	if task, ok := s.getTask(id); ok {
		snap := task.snapshot()
		if snap.Kind == FingerprintTaskKindAudit && snap.Status != FingerprintStatusRunning {
			// 已完成：尽量带上报告文件；读失败时至少返回任务状态。
			if rep, err := s.store.getAuditReport(id); err == nil {
				return &snap, rep, nil
			}
		}
		return &snap, nil, nil
	}
	// 内存没有（可能服务已重启）：查报告文件。
	rep, err := s.store.getAuditReport(id)
	if err != nil {
		return nil, nil, err
	}
	return nil, rep, nil
}

// ListAudits 检测记录列表：扫 audits 目录按时间倒序，前面合并内存中尚未落盘的任务。
func (s *FingerprintService) ListAudits() ([]*FingerprintAuditSummary, error) {
	summaries, err := s.store.listAuditReports()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(summaries))
	for _, sum := range summaries {
		seen[sum.ID] = struct{}{}
	}
	s.mu.Lock()
	pending := make([]*FingerprintAuditSummary, 0)
	for id, task := range s.tasks {
		if task.status.Kind != FingerprintTaskKindAudit {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		snap := task.snapshot()
		pending = append(pending, &FingerprintAuditSummary{
			ID:             snap.TaskID,
			ReferenceModel: snap.ReferenceModel,
			Status:         snap.Status,
			Progress:       snap.Progress,
			Flags:          []string{},
			Error:          snap.Error,
			CreatedAt:      snap.CreatedAt,
			Target:         task.target,
		})
	}
	s.mu.Unlock()
	out := append(pending, summaries...)
	if out == nil {
		out = []*FingerprintAuditSummary{}
	}
	return out, nil
}

// ListReferences 参考指纹列表（扫 references 目录，按注册时间倒序）。
func (s *FingerprintService) ListReferences() ([]*FingerprintReference, error) {
	return s.store.listReferences()
}

// DeleteReference 删除参考指纹文件（model 走与写文件相同的 slug 规则）。
func (s *FingerprintService) DeleteReference(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return ErrFingerprintReferenceNotFound
	}
	return s.store.deleteReference(model)
}

// DeleteAudit 删除检测报告文件；running 中的任务拒绝删除（409）。
func (s *FingerprintService) DeleteAudit(id string) error {
	if task, ok := s.getTask(id); ok && task.snapshot().Status == FingerprintStatusRunning {
		return ErrFingerprintAuditRunning
	}
	if err := s.store.deleteAuditReport(id); err != nil {
		return err
	}
	// 顺带清理内存任务（完成/失败状态），避免 GET 再查到。
	s.mu.Lock()
	delete(s.tasks, id)
	s.mu.Unlock()
	return nil
}

// ---------- 目标解析 ----------

// fingerprintProbeTarget 一次电池的探测目标。apiKey 只在任务运行期持有，不写文件/日志。
type fingerprintProbeTarget struct {
	provider       string // openai / anthropic / gemini / grok
	apiMode        string // openai 专用：chat_completions / responses
	baseURL        string
	apiKey         string
	model          string
	anthropicOAuth bool // anthropic + access_token：走 Bearer + anthropic-beta

	// Codex OAuth（ChatGPT 订阅）路径：固定走 chatgpt.com 内部 Codex 端点。
	codexOAuth       bool
	chatgptAccountID string // chatgpt-account-id 头（可为空）
	customUserAgent  string // 账号自定义 UA（空则用 codex CLI UA）

	// reasoningRejected 上游拒绝关闭 reasoning（400 且错误提到 reasoning）→ 该模型不适用单 token 指纹。
	reasoningRejected atomic.Bool
	// geminiThinkingUnsupported 上游不认 thinkingConfig → 后续请求省略该字段（§6.2：不支持则忽略）。
	geminiThinkingUnsupported atomic.Bool
	// codexReasoningUnsupported Codex 拒绝 reasoning 字段 → 后续请求省略（省略，不判不适用）。
	codexReasoningUnsupported atomic.Bool
	// maxOutputTokensUnsupported 上游（Codex/Responses）拒绝 max_output_tokens 字段
	// （400 且错误提到 max_output_tokens）→ 后续请求省略该字段，本次允许重试。
	// 与真实转发 openai_gateway_forward 的 rejected-field retry 行为一致；省略，不判不适用。
	maxOutputTokensUnsupported atomic.Bool
}

// resolveTarget 按 target_type 分发到账号/外部端点解析。
func (s *FingerprintService) resolveTarget(ctx context.Context, params FingerprintAuditParams) (*fingerprintProbeTarget, FingerprintReportTarget, error) {
	switch params.TargetType {
	case FingerprintTargetTypeAccount:
		if params.AccountID <= 0 {
			return nil, FingerprintReportTarget{}, ErrFingerprintMissingAccount
		}
		return s.resolveAccountTarget(ctx, params.AccountID, params.Model)
	case FingerprintTargetTypeExternal:
		return s.resolveExternalTarget(params)
	default:
		return nil, FingerprintReportTarget{}, ErrFingerprintInvalidTargetType
	}
}

// resolveAccountTarget 用系统内账号的凭证与平台类型构造探测目标。
// antigravity / composite 不支持（输出不是纯模型先验），返回明确错误。
func (s *FingerprintService) resolveAccountTarget(ctx context.Context, accountID int64, model string) (*fingerprintProbeTarget, FingerprintReportTarget, error) {
	account, err := s.accountSvc.GetByID(ctx, accountID)
	if err != nil {
		return nil, FingerprintReportTarget{}, err
	}

	target := &fingerprintProbeTarget{model: model, apiMode: MonitorAPIModeChatCompletions}
	switch account.Platform {
	case PlatformAnthropic:
		target.provider = MonitorProviderAnthropic
		target.baseURL = account.GetBaseURL() // 仅 apikey 账号有值（自带官方默认值）
		if target.baseURL == "" {
			target.baseURL = anthropicDefaultBaseURL
		}
	case PlatformOpenAI:
		target.provider = MonitorProviderOpenAI
		// Codex OAuth 订阅账号（无 api_key、只有 access_token）：必须走
		// chatgpt.com 内部 Codex 端点 + 专用身份头，普通 api.openai.com 请求会全灭。
		// 构造细节见 fingerprint_codex.go（参照账号测试连接 account_test_service）。
		if account.IsOAuth() && account.GetCredential("api_key") == "" && account.GetCredential("access_token") != "" {
			target.codexOAuth = true
			target.apiMode = MonitorAPIModeResponses
			target.model = normalizeCodexModel(model)
			target.chatgptAccountID = account.GetChatGPTAccountID()
			target.customUserAgent = strings.TrimSpace(account.GetOpenAIUserAgent())
		} else {
			target.baseURL = account.GetOpenAIBaseURL()
		}
	case PlatformGemini:
		target.provider = MonitorProviderGemini
		target.baseURL = account.GetGeminiBaseURL(geminiDefaultBaseURL)
	case PlatformGrok:
		target.provider = MonitorProviderGrok
		target.baseURL = account.GetGrokBaseURL()
	default:
		return nil, FingerprintReportTarget{}, ErrFingerprintUnsupportedPlatform
	}

	// 凭证：api_key 优先，空则 access_token（OAuth 账号；codexOAuth 路径同用此 access_token）。
	target.apiKey = account.GetCredential("api_key")
	if target.apiKey == "" {
		target.apiKey = account.GetCredential("access_token")
		target.anthropicOAuth = account.Platform == PlatformAnthropic
	}
	if target.apiKey == "" {
		return nil, FingerprintReportTarget{}, ErrFingerprintMissingCredential
	}

	reportTarget := FingerprintReportTarget{
		Type:      FingerprintTargetTypeAccount,
		AccountID: &accountID,
		Provider:  target.provider,
		Model:     model,
	}
	return target, reportTarget, nil
}

// resolveExternalTarget 构造外部端点探测目标。base_url 过 SSRF 校验（https + 公网 host），
// 防该功能被滥用打内网；api_key 仅运行期持有。
func (s *FingerprintService) resolveExternalTarget(params FingerprintAuditParams) (*fingerprintProbeTarget, FingerprintReportTarget, error) {
	provider := strings.TrimSpace(params.Provider)
	baseURL := strings.TrimSpace(params.BaseURL)
	apiKey := strings.TrimSpace(params.APIKey)
	if baseURL == "" || apiKey == "" || provider == "" {
		return nil, FingerprintReportTarget{}, ErrFingerprintMissingExternal
	}
	if !isSupportedProvider(provider) {
		return nil, FingerprintReportTarget{}, ErrFingerprintInvalidProvider
	}
	if err := validateAPIMode(provider, params.APIMode); err != nil {
		return nil, FingerprintReportTarget{}, err
	}
	if err := validateFingerprintExternalURL(baseURL); err != nil {
		return nil, FingerprintReportTarget{}, err
	}
	target := &fingerprintProbeTarget{
		provider: provider,
		apiMode:  defaultAPIMode(params.APIMode),
		baseURL:  baseURL,
		apiKey:   apiKey,
		model:    params.Model,
	}
	reportTarget := FingerprintReportTarget{
		Type:     FingerprintTargetTypeExternal,
		BaseURL:  baseURL,
		Provider: provider,
		Model:    params.Model,
	}
	return target, reportTarget, nil
}

// validateFingerprintExternalURL 校验外部 base_url：
// https scheme + host 非私网/loopback（与 validateEndpoint 同源，但允许携带 path，
// 因为常见中转 base_url 带 /v1 后缀）。
func validateFingerprintExternalURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ErrFingerprintInvalidEndpoint
	}
	ctx, cancel := context.WithTimeout(context.Background(), monitorEndpointResolveTimeout)
	defer cancel()
	blocked, err := isPrivateOrLoopbackHost(ctx, u.Hostname())
	if err != nil {
		return ErrFingerprintEndpointUnreachable
	}
	if blocked {
		return ErrFingerprintEndpointPrivate
	}
	return nil
}

// ---------- 任务执行 ----------

// executeAudit 后台执行：可选先注册参考 → 对被测目标跑电池 → 评分 → 写报告文件。
func (s *FingerprintService) executeAudit(task *fingerprintTask, target *fingerprintProbeTarget, reportTarget FingerprintReportTarget, refTarget *fingerprintProbeTarget, refAccountID *int64, loadedRef *FingerprintReference, params FingerprintAuditParams, exec fingerprintExecConfig) {
	defer func() {
		if r := recover(); r != nil {
			task.finish(FingerprintStatusFailed, fmt.Errorf("panic: %v", r))
		}
	}()
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), fingerprintBatteryTimeout)
	defer cancel()
	onProgress := func() { task.incProgress() }

	// 参考基准：现场注册（写 references 文件）或复用已加载的参考。
	reference := loadedRef
	if refTarget != nil {
		refResults, refLastErr := s.runBattery(ctx, refTarget, exec, onProgress)
		// 采样质量不达标：不写参考（保留同名旧文件），整个任务失败，不再测目标。
		if err := s.finishReferenceRegistration(params.ReferenceModel, refAccountID, refResults, refLastErr); err != nil {
			task.finish(FingerprintStatusFailed, err)
			return
		}
		reference = buildFingerprintReference(params.ReferenceModel, refAccountID, refResults)
	}

	results, lastErr := s.runBattery(ctx, target, exec, onProgress)
	cells, score, verdict, band, k, avgN, splitHalf, t0Mismatch, flags := scoreFingerprintBattery(results, reference, params.KeepRaw)
	// reasoning 关不掉 → 整个模型不适用单 token 指纹：标记 not_applicable 且不硬出判定（§8）。
	if target.reasoningRejected.Load() {
		flags = appendFingerprintFlag(flags, FingerprintFlagNotApplicable)
		verdict, band, score = FingerprintVerdictInsufficient, "", nil
	}

	report := &FingerprintReport{
		ID:     task.snapshot().TaskID,
		Target: reportTarget,
		Reference: FingerprintReportReference{
			Model:      reference.Model,
			Source:     reference.Source,
			EnrolledAt: reference.EnrolledAt,
		},
		Status:          FingerprintStatusDone,
		Progress:        task.snapshot().Progress,
		Score:           score,
		Verdict:         verdict,
		Band:            band,
		CellCount:       k,
		AvgSamples:      math.Round(avgN*100) / 100,
		SplitHalfJSD:    splitHalf,
		T0MismatchCells: t0Mismatch,
		Flags:           flags,
		LastError:       lastErr,
		Concurrency:     exec.Concurrency,
		IntervalMs:      exec.IntervalMs,
		CreatedBy:       params.OperatorID,
		CreatedAt:       task.snapshot().CreatedAt,
		DurationMs:      time.Since(start).Milliseconds(),
		Cells:           cells,
	}
	if err := s.store.saveAuditReport(report); err != nil {
		task.finish(FingerprintStatusFailed, fmt.Errorf("save audit report: %w", err))
		return
	}
	task.finish(FingerprintStatusDone, nil)
}

// executeReferenceRegistration 后台执行参考注册：电池 → 质量门槛 → 写 references/<model>.json。
// 采样质量不达标（≥10 有效样本的 cell 数 < 8）时任务失败且不写文件——
// 已有同名旧参考文件时保留旧的，避免空参考覆盖好参考。
func (s *FingerprintService) executeReferenceRegistration(task *fingerprintTask, target *fingerprintProbeTarget, accountID int64, model string, exec fingerprintExecConfig) {
	defer func() {
		if r := recover(); r != nil {
			task.finish(FingerprintStatusFailed, fmt.Errorf("panic: %v", r))
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), fingerprintBatteryTimeout)
	defer cancel()
	results, lastErr := s.runBattery(ctx, target, exec, func() { task.incProgress() })
	if err := s.finishReferenceRegistration(model, &accountID, results, lastErr); err != nil {
		task.finish(FingerprintStatusFailed, err)
		return
	}
	task.finish(FingerprintStatusDone, nil)
}

// finishReferenceRegistration 注册电池的收尾判定（纯逻辑，供单测）：
// 质量不达标返回错误且不写文件；达标才构建并写入参考。
func (s *FingerprintService) finishReferenceRegistration(model string, accountID *int64, results map[string]*fingerprintCellResult, lastErr string) error {
	if err := checkFingerprintReferenceQuality(results, lastErr); err != nil {
		return err
	}
	return s.store.saveReference(buildFingerprintReference(model, accountID, results))
}

// checkFingerprintReferenceQuality 注册质量门槛：≥fingerprintMinValidSamples 有效样本的
// cell 数达到 fingerprintMinCells 才允许写参考；否则返回含采样比例与最后上游错误的失败原因。
func checkFingerprintReferenceQuality(results map[string]*fingerprintCellResult, lastErr string) error {
	okCells, validSum, total := 0, 0, 0
	for _, res := range results {
		if res.valid >= fingerprintMinValidSamples {
			okCells++
		}
		validSum += res.valid
		total += res.valid + res.invalid + res.refusal + res.empty + res.failures
	}
	if okCells >= fingerprintMinCells {
		return nil
	}
	msg := fmt.Sprintf("采样质量不达标：%d/%d 个探测项获得 ≥%d 个有效样本（要求 ≥%d 项），有效样本 %d/%d",
		okCells, len(results), fingerprintMinValidSamples, fingerprintMinCells, validSum, total)
	if lastErr != "" {
		msg += "；最后上游错误：" + lastErr
	}
	return errors.New(msg)
}

// fingerprintReferenceEmpty 判断参考指纹是否为空（所有 cell 的 valid 均为 0）——
// 通常是注册时上游全部失败留下的坏文件。
func fingerprintReferenceEmpty(ref *FingerprintReference) bool {
	if ref == nil || len(ref.Cells) == 0 {
		return true
	}
	for _, c := range ref.Cells {
		if c != nil && c.Valid > 0 {
			return false
		}
	}
	return true
}

// ---------- 电池执行 ----------

// fingerprintCellResult 一个 cell 的全部采样结果（T=1.0 进分布，T=0 只留答案序列）。
type fingerprintCellResult struct {
	counts           map[string]int // valid 归一化答案分布
	samples          []string       // valid token 序列（劈半自校准用）
	rawSamples       []string       // 原始回答（keep_raw=true 时进报告）
	valid            int
	invalid          int
	refusal          int
	empty            int
	failures         int      // 重试后仍失败的探测数（不入数据）
	t0Answers        []string // T=0 归一化答案
	latenciesMs      []int    // T=1.0 成功样本延迟（response_caching 判据）
	completionTokens []int    // T=1.0 成功样本输出 token 数（hidden_reasoning 判据）
}

func newFingerprintCellResult() *fingerprintCellResult {
	return &fingerprintCellResult{counts: make(map[string]int)}
}

// runBattery 对目标跑完整探测电池：16 cell × (15 次 T=1.0 + 2 次 T=0)，
// cell 顺序随机打乱；worker 数与请求间隔由 exec 控制（所有 worker 共享一个节奏控制器），
// 单探测失败指数退避重试，失败样本不计入数据。
// 返回各 cell 结果与最后一次探测失败的摘要（已脱敏；无失败为空串）。
func (s *FingerprintService) runBattery(ctx context.Context, target *fingerprintProbeTarget, exec fingerprintExecConfig, onProgress func()) (map[string]*fingerprintCellResult, string) {
	rng := newFingerprintRNG()
	cells := fingerprintShuffledCells(rng)
	results := make(map[string]*fingerprintCellResult, len(cells))
	for _, c := range cells {
		results[c.Key()] = newFingerprintCellResult()
	}
	// 共享节奏控制：任意两个请求的发起间隔 ≥ exec.IntervalMs。
	pacer := &fingerprintPacer{interval: time.Duration(exec.IntervalMs) * time.Millisecond}

	type probeJob struct {
		cell        fingerprintCell
		prompt      string // 入队时抽好（rand.Rand 非并发安全）
		temperature float64
	}
	jobs := make(chan probeJob)
	var resultsMu sync.Mutex
	var wg sync.WaitGroup
	lastErr := ""

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			outcome, err := target.runProbe(ctx, j.prompt, j.temperature, pacer)
			resultsMu.Lock()
			res := results[j.cell.Key()]
			switch {
			case err != nil:
				// 失败样本不计入数据（§5）。
				res.failures++
				lastErr = err.Error()
			case j.temperature == fingerprintGreedyTemperature:
				if token, validity := normalizeFingerprintAnswer(outcome.text); validity == FingerprintValidityValid {
					res.t0Answers = append(res.t0Answers, token)
				}
			default:
				token, validity := normalizeFingerprintAnswer(outcome.text)
				switch validity {
				case FingerprintValidityValid:
					res.valid++
					res.counts[token]++
					res.samples = append(res.samples, token)
				case FingerprintValidityInvalid:
					res.invalid++
				case FingerprintValidityRefusal:
					res.refusal++
				default:
					res.empty++
				}
				res.rawSamples = append(res.rawSamples, outcome.text)
				res.latenciesMs = append(res.latenciesMs, outcome.latencyMs)
				res.completionTokens = append(res.completionTokens, outcome.completionTokens)
			}
			resultsMu.Unlock()
			if onProgress != nil {
				onProgress()
			}
		}
	}
	for i := 0; i < exec.Concurrency; i++ {
		wg.Add(1)
		go worker()
	}
	for _, c := range cells {
		for i := 0; i < fingerprintSamplesPerCell; i++ {
			jobs <- probeJob{cell: c, prompt: GenerateProbe(rng, c.Task, c.Language), temperature: fingerprintProbeTemperature}
		}
		for i := 0; i < fingerprintGreedySamplesPerCell; i++ {
			jobs <- probeJob{cell: c, prompt: GenerateProbe(rng, c.Task, c.Language), temperature: fingerprintGreedyTemperature}
		}
	}
	close(jobs)
	wg.Wait()
	return results, lastErr
}

// ---------- 单次探测 ----------

// fingerprintProbeOutcome 单次探测成功结果。
type fingerprintProbeOutcome struct {
	text             string // 原始回答文本
	completionTokens int    // 输出 token 数（hidden_reasoning 判据）
	latencyMs        int
}

// fingerprintProbeError 单次探测失败：携带可重试标记与限流信息；message 已脱敏。
type fingerprintProbeError struct {
	message    string
	retryable  bool
	statusCode int           // 非 2xx 的 HTTP 状态码（429 用于限流退避）
	retryAfter time.Duration // 429 响应的 Retry-After（已按 120s 上限截断）
}

func (e *fingerprintProbeError) Error() string { return e.message }

// runProbe 单次探测（含退避重试，最多 fingerprintMaxRetries 次）。
// 常规失败走指数退避；429 有 Retry-After 按其值等待，否则用更长的退避（10s/30s）。
func (t *fingerprintProbeTarget) runProbe(ctx context.Context, prompt string, temperature float64, pacer *fingerprintPacer) (*fingerprintProbeOutcome, error) {
	var lastErr *fingerprintProbeError
	for attempt := 0; attempt <= fingerprintMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := fingerprintRetryBaseBackoff << (attempt - 1)
			if lastErr != nil && lastErr.statusCode == http.StatusTooManyRequests {
				backoff = fingerprintRateLimitRetryBaseBackoff * time.Duration(2*attempt-1)
				if lastErr.retryAfter > 0 {
					backoff = lastErr.retryAfter
				}
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
		outcome, perr := t.runProbeOnce(ctx, prompt, temperature, pacer)
		if perr == nil {
			return outcome, nil
		}
		lastErr = perr
		if !perr.retryable {
			return nil, perr
		}
	}
	return nil, lastErr
}

// runProbeOnce 发一次探测请求：先过共享节奏控制，再构造 provider 特定 body →
// postFingerprintRawJSON（SSRF 安全客户端）→ 提取文本与 usage。
func (t *fingerprintProbeTarget) runProbeOnce(ctx context.Context, prompt string, temperature float64, pacer *fingerprintPacer) (*fingerprintProbeOutcome, *fingerprintProbeError) {
	body, err := t.buildBody(prompt, temperature)
	if err != nil {
		return nil, &fingerprintProbeError{message: err.Error()}
	}
	reqCtx, cancel := context.WithTimeout(ctx, fingerprintProbeTimeout)
	defer cancel()
	// 节奏控制计入请求超时预算：等待时隙也算发起前耗时。
	if err := pacer.wait(reqCtx); err != nil {
		return nil, &fingerprintProbeError{message: err.Error()}
	}

	start := time.Now()
	respBytes, status, respHeader, err := postFingerprintRawJSON(reqCtx, t.requestURL(), body, t.headers())
	latencyMs := int(time.Since(start) / time.Millisecond)
	if err != nil {
		// 网络/超时错误：可重试。
		return nil, &fingerprintProbeError{message: sanitizeErrorMessage(err.Error()), retryable: true}
	}
	if status < 200 || status >= 300 {
		snippet := sanitizeErrorMessage(truncateForErrorBody(string(respBytes)))
		perr := &fingerprintProbeError{message: fmt.Sprintf("upstream HTTP %d: %s", status, snippet), statusCode: status}
		if status == http.StatusTooManyRequests || status >= 500 {
			perr.retryable = true
		}
		if status == http.StatusTooManyRequests {
			if d, ok := parseFingerprintRetryAfter(respHeader); ok {
				perr.retryAfter = d
			}
		}
		lower := strings.ToLower(snippet)
		switch {
		case status == http.StatusBadRequest && strings.Contains(lower, "max_output_tokens") &&
			(t.codexOAuth || (t.provider == MonitorProviderOpenAI && t.apiMode == MonitorAPIModeResponses)):
			// Codex/Responses：上游 schema 窄不认 max_output_tokens 字段（仅这两个分支的请求体携带该字段）。
			// 与真实转发 openai_gateway_forward 的 rejected-field retry 一致：后续省略该字段，本次允许重试
			// （省略，不判不适用；一词约束在用户消息内，输出不受限后模型仍应遵守）。
			t.maxOutputTokensUnsupported.Store(true)
			perr.retryable = true
			// 上游一次拒绝多个非法字段时（如同时列出 max_output_tokens 与 reasoning），
			// 一并置位 reasoning 降级，避免重试循环里 reasoning case 永远轮不到而耗尽预算。
			if t.codexOAuth && (strings.Contains(lower, "reasoning") || strings.Contains(lower, "effort")) {
				t.codexReasoningUnsupported.Store(true)
			}
		case t.codexOAuth && status == http.StatusBadRequest &&
			(strings.Contains(lower, "reasoning") || strings.Contains(lower, "effort")):
			// Codex：reasoning 字段被拒 → 后续请求省略该字段，本次允许重试（省略，不判不适用）。
			t.codexReasoningUnsupported.Store(true)
			perr.retryable = true
		case (t.provider == MonitorProviderOpenAI || t.provider == MonitorProviderGrok) &&
			status == http.StatusBadRequest && strings.Contains(lower, "reasoning"):
			// openai/grok：上游拒绝关 reasoning → 模型不适用，不重试（§8）。
			t.reasoningRejected.Store(true)
		case t.provider == MonitorProviderGemini && status == http.StatusBadRequest && strings.Contains(lower, "thinking"):
			// gemini：thinkingConfig 不支持 → 后续请求省略该字段，本次允许重试（§6.2：不支持则忽略）。
			t.geminiThinkingUnsupported.Store(true)
			perr.retryable = true
		}
		return nil, perr
	}
	// Codex 是流式响应：从 SSE 里取 response.completed 的完整 response 对象。
	if t.codexOAuth {
		text, tokens, serr := extractFingerprintCodexSSE(respBytes)
		if serr != "" {
			return nil, &fingerprintProbeError{message: sanitizeErrorMessage(serr), retryable: true}
		}
		return &fingerprintProbeOutcome{text: text, completionTokens: tokens, latencyMs: latencyMs}, nil
	}
	text, tokens := extractFingerprintProbeResult(t.provider, t.apiMode, respBytes)
	return &fingerprintProbeOutcome{text: text, completionTokens: tokens, latencyMs: latencyMs}, nil
}

// buildBody 构造探测请求体：temperature 由调用方给（T=1.0 / T=0），max tokens=16，
// stream=false，系统提示强制一词回答；OpenAI 系关 reasoning，Gemini 关 thinking。
// Codex OAuth 走独立的流式构造（fingerprint_codex.go）。
func (t *fingerprintProbeTarget) buildBody(prompt string, temperature float64) ([]byte, error) {
	if t.codexOAuth {
		return buildCodexFingerprintBody(t, prompt, temperature)
	}
	switch {
	case t.provider == MonitorProviderOpenAI && t.apiMode == MonitorAPIModeResponses:
		body := map[string]any{
			"model":        t.model,
			"instructions": fingerprintSystemPrompt,
			"input":        prompt,
			"temperature":  temperature,
			"stream":       false,
			"reasoning":    map[string]any{"effort": "none"},
		}
		// 上游拒绝该字段后省略（见 runProbeOnce 的 rejected-field 重试），避免持续 400。
		if !t.maxOutputTokensUnsupported.Load() {
			body["max_output_tokens"] = fingerprintMaxTokens
		}
		return json.Marshal(body)
	case t.provider == MonitorProviderOpenAI || t.provider == MonitorProviderGrok:
		return json.Marshal(map[string]any{
			"model": t.model,
			"messages": []map[string]string{
				{"role": "system", "content": fingerprintSystemPrompt},
				{"role": "user", "content": prompt},
			},
			"temperature":      temperature,
			"max_tokens":       fingerprintMaxTokens,
			"stream":           false,
			"reasoning_effort": "none",
		})
	case t.provider == MonitorProviderAnthropic:
		// Anthropic 不加 thinking 字段即默认关闭（§6.2）。
		return json.Marshal(map[string]any{
			"model":       t.model,
			"system":      fingerprintSystemPrompt,
			"messages":    []map[string]string{{"role": "user", "content": prompt}},
			"max_tokens":  fingerprintMaxTokens,
			"temperature": temperature,
			"stream":      false,
		})
	case t.provider == MonitorProviderGemini:
		generationConfig := map[string]any{
			"maxOutputTokens": fingerprintMaxTokens,
			"temperature":     temperature,
		}
		if !t.geminiThinkingUnsupported.Load() {
			generationConfig["thinkingConfig"] = map[string]any{"thinkingBudget": 0}
		}
		return json.Marshal(map[string]any{
			"systemInstruction": map[string]any{"parts": []map[string]string{{"text": fingerprintSystemPrompt}}},
			"contents":          []map[string]any{{"parts": []map[string]string{{"text": prompt}}}},
			"generationConfig":  generationConfig,
		})
	}
	return nil, fmt.Errorf("unsupported provider %q", t.provider)
}

// headers 构造鉴权头。Anthropic OAuth（access_token）走 Bearer + anthropic-beta，
// 与账号测试连接的现有用法一致；api_key 走 x-api-key；Codex OAuth 走专用身份头。
func (t *fingerprintProbeTarget) headers() map[string]string {
	if t.codexOAuth {
		return codexFingerprintHeaders(t)
	}
	switch t.provider {
	case MonitorProviderAnthropic:
		if t.anthropicOAuth {
			return map[string]string{
				"Authorization":     "Bearer " + t.apiKey,
				"anthropic-version": monitorAnthropicAPIVersion,
				"anthropic-beta":    claude.BetaOAuth,
			}
		}
		return map[string]string{
			"x-api-key":         t.apiKey,
			"anthropic-version": monitorAnthropicAPIVersion,
		}
	case MonitorProviderGemini:
		// 与监控模块一致：x-goog-api-key header，避免 key 进 URL 被 *url.Error 回填到日志。
		return map[string]string{"x-goog-api-key": t.apiKey}
	default:
		return map[string]string{"Authorization": "Bearer " + t.apiKey}
	}
}

// requestURL 拼完整请求 URL：base_url 已含版本段（/v1、/v1beta）时不再重复拼接，
// 兼容中转站常见的带后缀 base_url。Codex OAuth 固定打 chatgpt.com 内部端点。
func (t *fingerprintProbeTarget) requestURL() string {
	if t.codexOAuth {
		return chatgptCodexAPIURL
	}
	base := strings.TrimRight(strings.TrimSpace(t.baseURL), "/")
	var version, rel string
	switch {
	case t.provider == MonitorProviderOpenAI && t.apiMode == MonitorAPIModeResponses:
		version, rel = "v1", "responses"
	case t.provider == MonitorProviderAnthropic:
		version, rel = "v1", "messages"
	case t.provider == MonitorProviderGemini:
		version, rel = "v1beta", "models/"+url.PathEscape(t.model)+":generateContent"
	default: // openai chat / grok
		version, rel = "v1", "chat/completions"
	}
	if strings.HasSuffix(strings.ToLower(base), "/"+version) {
		return base + "/" + rel
	}
	return base + "/" + version + "/" + rel
}

// extractFingerprintProbeResult 从响应 JSON 提取回答文本与输出 token 数。
func extractFingerprintProbeResult(provider, apiMode string, respBytes []byte) (string, int) {
	switch {
	case provider == MonitorProviderOpenAI && apiMode == MonitorAPIModeResponses:
		return extractOpenAIResponsesText(respBytes), int(gjson.GetBytes(respBytes, "usage.output_tokens").Int())
	case provider == MonitorProviderAnthropic:
		return extractAnthropicMonitorText(respBytes), int(gjson.GetBytes(respBytes, "usage.output_tokens").Int())
	case provider == MonitorProviderGemini:
		// content.parts 可能多段，拼接全部 text part。
		var b strings.Builder
		gjson.GetBytes(respBytes, "candidates.0.content.parts").ForEach(func(_, part gjson.Result) bool {
			_, _ = b.WriteString(part.Get("text").String())
			return true
		})
		return b.String(), int(gjson.GetBytes(respBytes, "usageMetadata.candidatesTokenCount").Int())
	default: // openai chat / grok
		return gjson.GetBytes(respBytes, "choices.0.message.content").String(),
			int(gjson.GetBytes(respBytes, "usage.completion_tokens").Int())
	}
}

// ---------- 参考构造与评分 ----------

// buildFingerprintReference 把一轮电池结果转成参考指纹文件结构。
func buildFingerprintReference(model string, accountID *int64, results map[string]*fingerprintCellResult) *FingerprintReference {
	ref := &FingerprintReference{
		Model:           model,
		Source:          FingerprintReferenceSourceAccountSampled,
		SourceAccountID: accountID,
		EnrolledAt:      time.Now().UTC(),
		Cells:           make(map[string]*FingerprintReferenceCell, len(results)),
	}
	for key, res := range results {
		ref.Cells[key] = &FingerprintReferenceCell{
			Samples:      res.valid + res.invalid + res.refusal + res.empty,
			Valid:        res.valid,
			Distribution: res.counts,
			T0Answers:    res.t0Answers,
		}
	}
	return ref
}

// scoreFingerprintBattery 汇总电池结果：逐 cell JSD → 平均 s、劈半自校准、
// 异常 flags（§8，被判掉的 cell 不进 JSD）、§7.2 分档 verdict。
func scoreFingerprintBattery(results map[string]*fingerprintCellResult, reference *FingerprintReference, keepRaw bool) (
	cells []*FingerprintReportCell, score *float64, verdict, band string,
	k int, avgN float64, splitHalf *float64, t0MismatchCells int, flags []string,
) {
	// 全局延迟中位数（response_caching 判据的分母）。
	var allLatencies []int
	for _, res := range results {
		allLatencies = append(allLatencies, res.latenciesMs...)
	}
	globalMedianLatency := fingerprintMedianInt(allLatencies)

	flagSet := make(map[string]struct{})
	jsdValues := make([]float64, 0, len(results))
	splitHalfValues := make([]float64, 0, len(results))
	validSum := 0

	// 固定 cell 顺序输出报告，便于阅读与 diff。
	for _, cell := range fingerprintCells() {
		res := results[cell.Key()]
		var refCell *FingerprintReferenceCell
		if reference != nil {
			refCell = reference.Cells[cell.Key()]
		}

		reportCell := &FingerprintReportCell{
			Task:     cell.Task,
			Language: cell.Language,
			Valid:    res.valid,
			Invalid:  res.invalid,
			Refusal:  res.refusal,
			Empty:    res.empty,
		}
		if len(res.counts) > 0 {
			reportCell.TopAnswers = res.counts
		}
		if refCell != nil && len(refCell.Distribution) > 0 {
			reportCell.ReferenceTopAnswers = refCell.Distribution
		}
		if len(res.t0Answers) > 0 {
			reportCell.T0Answers = res.t0Answers
		}
		if keepRaw && len(res.rawSamples) > 0 {
			reportCell.Samples = res.rawSamples
		}

		// §8 异常筛查 1：T=1.0 分布坍缩（单一答案）且延迟中位数低于全局一半以上 → 响应级缓存。
		if res.valid >= fingerprintMinValidSamples && len(res.counts) == 1 &&
			globalMedianLatency > 0 && fingerprintMedianInt(res.latenciesMs) < globalMedianLatency/2 {
			reportCell.Excluded = fingerprintExcludedResponseCaching
			flagSet[FingerprintFlagResponseCaching] = struct{}{}
		}
		// §8 异常筛查 2：一词回答的 completion_tokens 持续打满/超过 16 → 隐藏 reasoning，不可审计。
		// （≥ 判定而非 >：隐藏 reasoning 的模型会把 max_output_tokens=16 打满，
		// 正常一词回答的输出 token 通常只有 1–5，不会触到上限。）
		if reportCell.Excluded == "" && len(res.completionTokens) >= fingerprintSplitHalfMinSamples &&
			fingerprintMedianInt(res.completionTokens) >= fingerprintMaxTokens {
			reportCell.Excluded = fingerprintExcludedHiddenReasoning
			flagSet[FingerprintFlagHiddenReasoning] = struct{}{}
		}

		// 双侧 valid ≥ 10 才进 JSD（§7.1）；被判掉的 cell 不进。
		if reportCell.Excluded == "" {
			if refCell != nil && refCell.Valid >= fingerprintMinValidSamples && res.valid >= fingerprintMinValidSamples {
				jsdVal := fingerprintJSD(refCell.Distribution, res.counts)
				reportCell.JSD = &jsdVal
				jsdValues = append(jsdValues, jsdVal)
				validSum += res.valid
			} else {
				reportCell.Excluded = fingerprintExcludedInsufficientSamples
			}
		}

		// 劈半自校准（自身稳定性，与是否进 JSD 解耦）。
		if v, ok := fingerprintSplitHalfJSD(res.samples); ok {
			splitHalfValues = append(splitHalfValues, v)
		}
		// T=0 快信号：与参考的 T=0 首答案不一致 → 模型被更新/替换的即时提示（§7.2）。
		if refCell != nil && len(refCell.T0Answers) > 0 && len(res.t0Answers) > 0 &&
			res.t0Answers[0] != refCell.T0Answers[0] {
			t0MismatchCells++
		}
		cells = append(cells, reportCell)
	}

	k = len(jsdValues)
	if k > 0 {
		avgN = float64(validSum) / float64(k)
	}
	if s, ok := fingerprintMeanJSD(jsdValues); ok {
		score = &s
	}
	if v, ok := fingerprintMeanJSD(splitHalfValues); ok {
		splitHalf = &v
	}

	s := 0.0
	if score != nil {
		s = *score
	}
	verdict, band = fingerprintVerdictFor(s, k, avgN)

	flags = make([]string, 0, len(flagSet))
	for f := range flagSet {
		flags = append(flags, f)
	}
	sortFingerprintFlags(flags)
	return cells, score, verdict, band, k, avgN, splitHalf, t0MismatchCells, flags
}

// fingerprintMedianInt 整数序列中位数（偶数个取中间两值均值）；空序列返回 0。
func fingerprintMedianInt(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return float64(sorted[mid])
	}
	return float64(sorted[mid-1]+sorted[mid]) / 2
}

// appendFingerprintFlag 追加 flag（去重）。
func appendFingerprintFlag(flags []string, f string) []string {
	for _, existing := range flags {
		if existing == f {
			return flags
		}
	}
	return append(flags, f)
}

// sortFingerprintFlags 固定 flags 输出顺序，保证报告文件可 diff。
func sortFingerprintFlags(flags []string) {
	sort.Strings(flags)
}
