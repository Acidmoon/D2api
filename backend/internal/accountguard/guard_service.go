// Package accountguard 实现账号级内容违规守护：在网关选定上游账号之后、
// 首个上游请求发出之前，复用 prompt-audit 的 Qwen3Guard 端点池同步判定请求内容；
// 命中违规时计数，窗口内达到阈值则临时封禁账号并邮件告警管理员。
//
// 该包独立于 internal/service 存在是因为 securityaudit 已反向依赖 service
// （coordinator_legacy.go / prompt_config_store.go），service 直接 import
// securityaudit 会构成 import 环；service 层只依赖这里实现的
// service.AccountViolationGuard 接口。
package accountguard

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// violationNotifyCooldown 同一账号封禁告警邮件的最小间隔。
const violationNotifyCooldown = 30 * time.Minute

// notifyTimeout 异步告警的整体超时（与分组不可用告警保持一致）。
const notifyTimeout = 20 * time.Second

// AccountBanRepository 是封禁原语的最小依赖（由 service.AccountRepository 满足）。
type AccountBanRepository interface {
	SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error
}

// SettingValueReader 是设置读取的最小依赖（由 service.SettingRepository 满足）。
type SettingValueReader interface {
	GetValue(ctx context.Context, key string) (string, error)
}

// EmailSender 是邮件发送的最小依赖（由 *service.EmailService 满足）。
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// activeConfigStore 是 prompt-audit 生效配置读取的最小依赖
// （由 *securityaudit.ConfigManager 满足）。
type activeConfigStore interface {
	Active() (securityaudit.ActiveConfig, bool)
}

// promptGuardEvaluator 是同步评估器的最小依赖
// （由 *securityaudit.GuardEvaluator 满足，测试注入 fake）。
type promptGuardEvaluator interface {
	Evaluate(ctx context.Context, cfg securityaudit.ActiveConfig, snapshot securityaudit.PromptSnapshot) (*securityaudit.PromptDecision, error)
}

// GuardService 账号级内容违规守护服务，实现 service.AccountViolationGuard。
type GuardService struct {
	config       activeConfigStore
	evaluator    promptGuardEvaluator
	accountRepo  AccountBanRepository
	settings     SettingValueReader
	emailService EmailSender
	counter      service.ViolationCounterCache
	now          func() time.Time
}

// NewGuardService 创建账号违规守护服务。evaluator 复用 prompt-audit 的
// 同步评估器（调用方应注入不带事件落库 repo 的 GuardEvaluator，
// 避免与异步审计重复记录事件）。
func NewGuardService(
	config activeConfigStore,
	evaluator promptGuardEvaluator,
	accountRepo AccountBanRepository,
	settings SettingValueReader,
	emailService EmailSender,
	counter service.ViolationCounterCache,
) *GuardService {
	return &GuardService{
		config:       config,
		evaluator:    evaluator,
		accountRepo:  accountRepo,
		settings:     settings,
		emailService: emailService,
		counter:      counter,
		now:          time.Now,
	}
}

// Check 实现 service.AccountViolationGuard。
// 策略：审核不可用/超时/响应非法一律 fail-open（放行且不计数），
// 与 prompt-audit blocking 的 fail-closed 刻意不同——该调用点在账号选定之后，
// 阻断会影响已完成调度的真实流量，可用性优先。
func (s *GuardService) Check(ctx context.Context, input service.AccountGuardCheckInput) (*service.AccountGuardDecision, error) {
	if s == nil || s.config == nil || s.evaluator == nil || input.Account == nil {
		return nil, nil
	}
	cfg, ok := s.config.Active()
	if !ok || !cfg.AccountGuard.Enabled {
		return nil, nil
	}
	if len(cfg.EnabledEndpoints()) == 0 {
		return nil, nil
	}
	snapshot, err := securityaudit.ExtractBlockingPromptSnapshot(securityaudit.Request{
		RequestID: input.RequestID,
		UserID:    input.UserID,
		APIKeyID:  input.APIKeyID,
		GroupID:   cloneInt64Ptr(input.GroupID),
		Provider:  input.Account.Platform,
		Protocol:  input.Protocol,
		Model:     input.Model,
		Body:      input.Body,
		Stage:     "account_guard",
	}, cfg.BlockingLatestTurnOnly)
	if err != nil {
		// 无可扫描文本或请求体无法解析：放行，不计数。
		return nil, nil
	}
	decision, err := s.evaluator.Evaluate(ctx, cfg, snapshot)
	if err != nil || decision == nil {
		// 审核端点不可用/超时/响应非法：fail-open 放行，不计数。
		slog.Warn("account_violation_guard.evaluate_failed",
			"account_id", input.Account.ID,
			"err", err,
		)
		return nil, nil
	}
	if decision.Kind != securityaudit.DecisionBlock {
		// Allow / Flag（含 Controversial、未命中已启用分类的 Unsafe）不计数。
		return nil, nil
	}
	reason := violationReason(decision.Result)
	slog.Warn("account_violation_guard.violation",
		"account_id", input.Account.ID,
		"account_name", input.Account.Name,
		"platform", input.Account.Platform,
		"reason", reason,
	)
	s.recordViolation(ctx, input.Account, cfg.AccountGuard, reason)
	return &service.AccountGuardDecision{Blocked: true, Reason: reason}, nil
}

// recordViolation 计数并在达到阈值时封禁账号 + 异步告警。
// 计数器/封禁/邮件的任何失败都只记日志，不影响已经做出的阻断决定。
func (s *GuardService) recordViolation(ctx context.Context, account *service.Account, guardCfg securityaudit.AccountGuardConfig, reason string) {
	if s.counter == nil {
		return
	}
	window := time.Duration(guardCfg.WindowMinutes) * time.Minute
	count, err := s.counter.IncrementViolationCount(ctx, account.ID, window)
	if err != nil {
		slog.Warn("account_violation_guard.count_failed", "account_id", account.ID, "err", err)
		return
	}
	if count < int64(guardCfg.Threshold) {
		return
	}
	banDuration := time.Duration(guardCfg.BanDurationMinutes) * time.Minute
	until := s.now().Add(banDuration)
	banReason := fmt.Sprintf("account_violation_guard: %d 次内容违规 / %d 分钟窗口（%s）", count, guardCfg.WindowMinutes, reason)
	if s.accountRepo != nil {
		if err := s.accountRepo.SetTempUnschedulable(ctx, account.ID, until, banReason); err != nil {
			slog.Warn("account_violation_guard.ban_failed", "account_id", account.ID, "err", err)
			return
		}
	}
	// 封禁后清零计数，窗口在封禁结束恢复调度后重新起算。
	if err := s.counter.ResetViolationCount(ctx, account.ID); err != nil {
		slog.Warn("account_violation_guard.reset_failed", "account_id", account.ID, "err", err)
	}
	slog.Warn("account_violation_guard.account_banned",
		"account_id", account.ID,
		"account_name", account.Name,
		"platform", account.Platform,
		"count", count,
		"window_minutes", guardCfg.WindowMinutes,
		"ban_duration_minutes", guardCfg.BanDurationMinutes,
		"until", until,
	)
	s.NotifyAsync(ViolationNotifyInput{
		AccountID:          account.ID,
		AccountName:        account.Name,
		Platform:           account.Platform,
		ViolationCount:     count,
		Threshold:          guardCfg.Threshold,
		WindowMinutes:      guardCfg.WindowMinutes,
		BanDurationMinutes: guardCfg.BanDurationMinutes,
		BannedUntil:        until,
		Reason:             reason,
		OccurredAt:         s.now(),
	})
}

// ViolationNotifyInput 封禁告警邮件的内容参数。
type ViolationNotifyInput struct {
	AccountID          int64
	AccountName        string
	Platform           string
	ViolationCount     int64
	Threshold          int
	WindowMinutes      int
	BanDurationMinutes int
	BannedUntil        time.Time
	Reason             string
	OccurredAt         time.Time
}

// NotifyAsync 异步发送封禁告警邮件（goroutine + 20s 超时），按账号 30 分钟冷却。
func (s *GuardService) NotifyAsync(input ViolationNotifyInput) {
	if s == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
		defer cancel()
		if err := s.Notify(ctx, input); err != nil {
			slog.Warn("account_violation_guard.notify_failed",
				"account_id", input.AccountID,
				"err", err,
			)
		}
	}()
}

// Notify 同步发送封禁告警邮件；无收件人、冷却未结束或依赖缺失时静默跳过。
func (s *GuardService) Notify(ctx context.Context, input ViolationNotifyInput) error {
	if s == nil || s.emailService == nil || s.settings == nil {
		return nil
	}
	if input.AccountID <= 0 {
		return nil
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = s.now()
	}
	recipients := s.recipients(ctx)
	if len(recipients) == 0 {
		return nil
	}
	if s.counter != nil {
		claimed, err := s.counter.ClaimViolationNotifyCooldown(ctx, input.AccountID, violationNotifyCooldown)
		if err != nil {
			// 冷却判断失败时宁可不发，避免故障期间邮件轰炸。
			slog.Warn("account_violation_guard.cooldown_failed", "account_id", input.AccountID, "err", err)
			return nil
		}
		if !claimed {
			return nil
		}
	}
	subject := fmt.Sprintf("[D2api] 账号内容违规自动封禁：%s", firstNonEmpty(strings.TrimSpace(input.AccountName), fmt.Sprintf("Account #%d", input.AccountID)))
	body := buildViolationNotifyBody(input)
	for _, recipient := range recipients {
		if err := s.emailService.SendEmail(ctx, recipient, subject, body); err != nil {
			return err
		}
	}
	return nil
}

// recipients 复用分组不可用告警的收件人设置（group_unavailable_alert_emails），
// 只保留已验证且未禁用的邮箱。
func (s *GuardService) recipients(ctx context.Context) []string {
	raw, err := s.settings.GetValue(ctx, service.SettingKeyGroupUnavailableAlertEmails)
	if err != nil || strings.TrimSpace(raw) == "" || raw == "[]" {
		return nil
	}
	seen := make(map[string]struct{})
	var recipients []string
	for _, entry := range service.ParseNotifyEmails(raw) {
		if entry.Disabled || !entry.Verified {
			continue
		}
		email := strings.TrimSpace(entry.Email)
		if email == "" {
			continue
		}
		lower := strings.ToLower(email)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		recipients = append(recipients, email)
	}
	return recipients
}

func buildViolationNotifyBody(input ViolationNotifyInput) string {
	rows := []string{
		notifyTableRow("账号", fmt.Sprintf("%s (#%d)", firstNonEmpty(input.AccountName, "未命名账号"), input.AccountID)),
		notifyTableRow("平台", input.Platform),
		notifyTableRow("窗口内违规次数", fmt.Sprintf("%d（阈值 %d）", input.ViolationCount, input.Threshold)),
		notifyTableRow("统计窗口", fmt.Sprintf("%d 分钟", input.WindowMinutes)),
		notifyTableRow("封禁时长", fmt.Sprintf("%d 分钟（至 %s）", input.BanDurationMinutes, input.BannedUntil.Format(time.RFC3339))),
		notifyTableRow("违规类别", input.Reason),
		notifyTableRow("触发时间", input.OccurredAt.Format(time.RFC3339)),
	}
	return `<!doctype html><html><body style="font-family:Arial,sans-serif;line-height:1.5;color:#111827;">` +
		`<h2>D2api 账号内容违规自动封禁</h2>` +
		`<p>该账号在统计窗口内命中的内容违规次数达到阈值，已被临时移出调度，到期自动恢复。邮件使用系统 SMTP 配置发送。</p>` +
		`<table cellpadding="6" cellspacing="0" border="1" style="border-collapse:collapse;border-color:#d1d5db;">` +
		strings.Join(rows, "") +
		`</table></body></html>`
}

func notifyTableRow(label, value string) string {
	return fmt.Sprintf(
		`<tr><th align="left" style="background:#f3f4f6;">%s</th><td>%s</td></tr>`,
		html.EscapeString(label),
		html.EscapeString(firstNonEmpty(value, "-")),
	)
}

// violationReason 从 Guard 归一化结果中提取违规类别作为原因示例。
func violationReason(result *securityaudit.NormalizedResult) string {
	if result == nil {
		return "content_policy_violation"
	}
	categories := result.MatchedScanners
	if len(categories) == 0 {
		categories = result.Categories
	}
	if len(categories) == 0 {
		if result.Safety != "" {
			return strings.ToLower(result.Safety)
		}
		return "content_policy_violation"
	}
	return strings.Join(categories, ",")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
