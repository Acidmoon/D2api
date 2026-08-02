package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AccountViolationGuardBlockMessage 是账号违规守护阻断请求时返回给客户端的统一文案。
const AccountViolationGuardBlockMessage = "请求被内容安全策略拒绝，请调整输入后重试"

// AccountGuardCheckInput 账号级内容违规守护的一次检查输入。
// Body 必须是当前账号尝试对应的原始客户端请求体（未经账号级改写），
// Protocol 为空时由实现方按已知协议逐个尝试提取。
type AccountGuardCheckInput struct {
	Account   *Account
	Protocol  string
	Model     string
	Body      []byte
	GroupID   *int64
	APIKeyID  int64
	UserID    int64
	RequestID string
}

// AccountGuardDecision 账号违规守护的判定结果。
// Blocked 为 true 时调用方必须以 403 拒绝该请求，且不得 failover 到其他账号。
type AccountGuardDecision struct {
	Blocked bool
	// Reason 违规原因示例（如 Qwen3Guard 命中分类），用于日志与告警邮件。
	Reason string
}

// AccountViolationGuard 账号级内容违规守护接口。
// 实现位于 internal/accountguard（该包同时依赖 service 与 securityaudit，
// 接口定义在 service 层以避免 import 环）。
type AccountViolationGuard interface {
	// Check 在账号选定之后、首个上游请求发出之前同步判定请求内容。
	// 审核服务不可用/超时等错误必须 fail-open（放行且不计数），
	// 实现方不得把评估错误返回给调用方阻断流量。
	Check(ctx context.Context, input AccountGuardCheckInput) (*AccountGuardDecision, error)
}

// ViolationCounterCache 账号违规计数缓存接口（Redis 实现位于 repository 层）。
type ViolationCounterCache interface {
	// IncrementViolationCount 增加账号的违规计数，返回窗口内当前计数值。
	// window 是计数窗口时长，首次写入时设置过期时间，超过窗口计数器自动重置。
	IncrementViolationCount(ctx context.Context, accountID int64, window time.Duration) (int64, error)
	// ResetViolationCount 重置账号的违规计数（封禁后清零，窗口重新起算）。
	ResetViolationCount(ctx context.Context, accountID int64) error
	// ClaimViolationNotifyCooldown 以 SET NX EX 语义抢占账号的告警冷却窗口，
	// 返回 true 表示本次调用获得发送权（冷却期内其他调用返回 false）。
	ClaimViolationNotifyCooldown(ctx context.Context, accountID int64, cooldown time.Duration) (bool, error)
}

// AccountGuardBlockedError 账号违规守护阻断错误。
// 该类型刻意不是 UpstreamFailoverError：handler 仅对 UpstreamFailoverError
// 触发换号 failover，因此返回本错误可保证违规请求不会被重试到其他账号。
type AccountGuardBlockedError struct {
	AccountID int64
	Reason    string
}

func (e *AccountGuardBlockedError) Error() string {
	if e == nil {
		return "account guard blocked"
	}
	return fmt.Sprintf("account guard blocked: account %d violated content policy (%s)", e.AccountID, e.Reason)
}

// runAccountViolationGuard 是两个网关服务共用的守护检查逻辑：
// 调用实现方、组装输入、 fail-open 处理评估错误。
// 返回 nil 表示放行；返回非 nil decision 表示命中违规必须阻断。
func runAccountViolationGuard(guard AccountViolationGuard, ctx context.Context, c *gin.Context, account *Account, protocol, model string, body []byte) *AccountGuardDecision {
	if guard == nil || account == nil {
		return nil
	}
	input := AccountGuardCheckInput{
		Account:  account,
		Protocol: protocol,
		Model:    model,
		Body:     body,
	}
	if apiKey := getAPIKeyFromContext(c); apiKey != nil {
		input.APIKeyID = apiKey.ID
		input.UserID = apiKey.UserID
		input.GroupID = apiKey.GroupID
	}
	decision, err := guard.Check(ctx, input)
	if err != nil || decision == nil || !decision.Blocked {
		// 评估失败（审核端点不可用/超时/响应非法）按 fail-open 放行，
		// 避免新调用点引入可用性风险；错误已在实现方记录。
		return nil
	}
	return decision
}

// checkAccountViolationGuard OpenAI 网关路径的账号违规守护检查。
// 命中违规时直接写入 403（与 prompt-audit blocking 的阻断响应格式一致），
// 并返回 AccountGuardBlockedError——该错误不是 UpstreamFailoverError，
// handler 不会因此 failover 到其他账号。
func (s *OpenAIGatewayService) checkAccountViolationGuard(ctx context.Context, c *gin.Context, account *Account, protocol, model string, body []byte) error {
	decision := runAccountViolationGuard(s.accountGuard, ctx, c, account, protocol, model, body)
	if decision == nil {
		return nil
	}
	if c != nil {
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"type":    "permission_error",
			"code":    "content_policy_violation",
			"message": AccountViolationGuardBlockMessage,
		}})
	}
	return &AccountGuardBlockedError{AccountID: account.ID, Reason: decision.Reason}
}

// checkAccountViolationGuard Anthropic 网关路径的账号违规守护检查（语义同上，
// 响应体使用 Anthropic 错误信封格式）。
func (s *GatewayService) checkAccountViolationGuard(ctx context.Context, c *gin.Context, account *Account, protocol, model string, body []byte) error {
	decision := runAccountViolationGuard(s.accountGuard, ctx, c, account, protocol, model, body)
	if decision == nil {
		return nil
	}
	if c != nil {
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
		c.JSON(http.StatusForbidden, gin.H{"type": "error", "error": gin.H{
			"type":    "permission_error",
			"code":    "content_policy_violation",
			"message": AccountViolationGuardBlockMessage,
		}})
	}
	return &AccountGuardBlockedError{AccountID: account.ID, Reason: decision.Reason}
}
