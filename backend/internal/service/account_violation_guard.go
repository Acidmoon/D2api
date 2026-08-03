package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// UserViolationGuardBlockMessage 是用户违规守护阻断请求时返回给客户端的统一文案。
const UserViolationGuardBlockMessage = "请求被内容安全策略拒绝，请调整输入后重试"

// UserGuardCheckInput 用户级内容违规守护的一次检查输入。
// Body 必须是当前账号尝试对应的原始客户端请求体（未经账号级改写），
// Protocol 为空时由实现方按已知协议逐个尝试提取。
// 计数与封禁按 UserID 维度；UserID<=0（无用户上下文）时实现方放行不计数。
type UserGuardCheckInput struct {
	Account     *Account
	Protocol    string
	Model       string
	Body        []byte
	GroupID     *int64
	APIKeyID    int64
	UserID      int64
	Username    string
	UserEmail   string
	UserIsAdmin bool
	RequestID   string
}

// UserGuardDecision 用户违规守护的判定结果。
// Blocked 为 true 时调用方必须以 403 拒绝该请求，且不得 failover 到其他账号。
type UserGuardDecision struct {
	Blocked bool
	// Reason 违规原因示例（如 Qwen3Guard 命中分类），用于日志与告警邮件。
	Reason string
}

// UserViolationGuard 用户级内容违规守护接口。
// 实现位于 internal/accountguard（该包同时依赖 service 与 securityaudit，
// 接口定义在 service 层以避免 import 环）。
type UserViolationGuard interface {
	// Check 在账号选定之后、首个上游请求发出之前同步判定请求内容。
	// 审核服务不可用/超时等错误必须 fail-open（放行且不计数），
	// 实现方不得把评估错误返回给调用方阻断流量。
	Check(ctx context.Context, input UserGuardCheckInput) (*UserGuardDecision, error)
}

// ViolationCounterCache 用户违规计数/封禁缓存接口（Redis 实现位于 repository 层）。
type ViolationCounterCache interface {
	// IncrementViolationCount 增加用户的违规计数，返回窗口内当前计数值。
	// window 是计数窗口时长，首次写入时设置过期时间，超过窗口计数器自动重置。
	IncrementViolationCount(ctx context.Context, userID int64, window time.Duration) (int64, error)
	// ResetViolationCount 重置用户的违规计数（封禁后清零，窗口重新起算）。
	ResetViolationCount(ctx context.Context, userID int64) error
	// ClaimViolationNotifyCooldown 以 SET NX EX 语义抢占用户的告警冷却窗口，
	// 返回 true 表示本次调用获得发送权（冷却期内其他调用返回 false）。
	ClaimViolationNotifyCooldown(ctx context.Context, userID int64, cooldown time.Duration) (bool, error)
	// SetUserViolationBan 写入用户临时封禁键（值=解封时间，TTL=封禁时长，到期自动恢复）。
	SetUserViolationBan(ctx context.Context, userID int64, until time.Time, ttl time.Duration) error
	// GetUserViolationBan 读取用户临时封禁状态，返回 (解封时间, 是否封禁中)。
	GetUserViolationBan(ctx context.Context, userID int64) (time.Time, bool, error)
	// GetUserViolationBans 批量读取用户临时封禁状态（管理端用户列表用），
	// 返回仍在封禁中的 userID → 解封时间；未封禁/过期/非法键不出现。
	GetUserViolationBans(ctx context.Context, userIDs []int64) (map[int64]time.Time, error)
	// ClearUserViolationBan 删除用户临时封禁键（管理员解除封禁）。
	ClearUserViolationBan(ctx context.Context, userID int64) error
}

// UserGuardBlockedError 用户违规守护阻断错误。
// 该类型刻意不是 UpstreamFailoverError：handler 仅对 UpstreamFailoverError
// 触发换号 failover，因此返回本错误可保证违规请求不会被重试到其他账号。
type UserGuardBlockedError struct {
	AccountID int64
	Reason    string
}

func (e *UserGuardBlockedError) Error() string {
	if e == nil {
		return "account guard blocked"
	}
	return fmt.Sprintf("account guard blocked: account %d violated content policy (%s)", e.AccountID, e.Reason)
}

// runUserViolationGuard 是两个网关服务共用的守护检查逻辑：
// 调用实现方、组装输入、 fail-open 处理评估错误。
// 返回 nil 表示放行；返回非 nil decision 表示命中违规必须阻断。
func runUserViolationGuard(guard UserViolationGuard, ctx context.Context, c *gin.Context, account *Account, protocol, model string, body []byte) *UserGuardDecision {
	if guard == nil || account == nil {
		return nil
	}
	input := UserGuardCheckInput{
		Account:  account,
		Protocol: protocol,
		Model:    model,
		Body:     body,
	}
	if apiKey := getAPIKeyFromContext(c); apiKey != nil {
		input.APIKeyID = apiKey.ID
		input.UserID = apiKey.UserID
		input.GroupID = apiKey.GroupID
		if apiKey.User != nil {
			input.Username = apiKey.User.Username
			input.UserEmail = apiKey.User.Email
			input.UserIsAdmin = apiKey.User.IsAdmin()
		}
	}
	decision, err := guard.Check(ctx, input)
	if err != nil || decision == nil || !decision.Blocked {
		// 评估失败（审核端点不可用/超时/响应非法）按 fail-open 放行，
		// 避免新调用点引入可用性风险；错误已在实现方记录。
		return nil
	}
	return decision
}

// checkUserViolationGuard OpenAI 网关路径的用户违规守护检查。
// 命中违规时直接写入 403（与 prompt-audit blocking 的阻断响应格式一致），
// 并返回 UserGuardBlockedError——该错误不是 UpstreamFailoverError，
// handler 不会因此 failover 到其他账号。
func (s *OpenAIGatewayService) checkUserViolationGuard(ctx context.Context, c *gin.Context, account *Account, protocol, model string, body []byte) error {
	decision := runUserViolationGuard(s.userGuard, ctx, c, account, protocol, model, body)
	if decision == nil {
		return nil
	}
	if c != nil {
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"type":    "permission_error",
			"code":    "content_policy_violation",
			"message": UserViolationGuardBlockMessage,
		}})
	}
	return &UserGuardBlockedError{AccountID: account.ID, Reason: decision.Reason}
}

// checkUserViolationGuard Anthropic 网关路径的用户违规守护检查（语义同上，
// 响应体使用 Anthropic 错误信封格式）。
func (s *GatewayService) checkUserViolationGuard(ctx context.Context, c *gin.Context, account *Account, protocol, model string, body []byte) error {
	decision := runUserViolationGuard(s.userGuard, ctx, c, account, protocol, model, body)
	if decision == nil {
		return nil
	}
	if c != nil {
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
		c.JSON(http.StatusForbidden, gin.H{"type": "error", "error": gin.H{
			"type":    "permission_error",
			"code":    "content_policy_violation",
			"message": UserViolationGuardBlockMessage,
		}})
	}
	return &UserGuardBlockedError{AccountID: account.ID, Reason: decision.Reason}
}
