package middleware

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthGoogle is a Google-style error wrapper for API key auth.
func APIKeyAuthGoogle(apiKeyService *service.APIKeyService, cfg *config.Config) gin.HandlerFunc {
	return APIKeyAuthWithSubscriptionGoogle(apiKeyService, nil, cfg)
}

// APIKeyAuthWithSubscriptionGoogle behaves like ApiKeyAuthWithSubscription but returns Google-style errors:
// {"error":{"code":401,"message":"...","status":"UNAUTHENTICATED"}}
//
// It is intended for Gemini native endpoints (/v1beta) to match Gemini SDK expectations.
func APIKeyAuthWithSubscriptionGoogle(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rejectInvalidAuthAbuse(c, apiKeyService) {
			abortWithGoogleError(c, 429, "Too many invalid authentication attempts; retry later")
			return
		}
		if apiKeyHeadersTooLarge(c) {
			recordInvalidAuthFailure(c, apiKeyService)
			MarkIngressRejected(c, IngressRejectInvalidAPIKey)
			abortWithGoogleError(c, 401, "Invalid API key")
			return
		}
		if v := strings.TrimSpace(c.Query("api_key")); v != "" {
			recordInvalidAuthFailure(c, apiKeyService)
			MarkIngressRejected(c, IngressRejectQueryAPIKeyDeprecated)
			abortWithGoogleError(c, 400, "Query parameter api_key is deprecated. Use Authorization header or key instead.")
			return
		}
		apiKeyString := extractAPIKeyForGoogle(c)
		if apiKeyString == "" {
			recordInvalidAuthFailure(c, apiKeyService)
			if hasAPIKeyCredentialInput(c) {
				MarkIngressRejected(c, IngressRejectInvalidAPIKey)
			} else {
				MarkIngressRejected(c, IngressRejectAPIKeyRequired)
			}
			abortWithGoogleError(c, 401, "API key is required")
			return
		}
		if len(apiKeyString) > service.MaxAPIKeyCredentialBytes {
			recordInvalidAuthFailure(c, apiKeyService)
			MarkIngressRejected(c, IngressRejectInvalidAPIKey)
			abortWithGoogleError(c, 401, "Invalid API key")
			return
		}

		apiKey, err := apiKeyService.GetByKey(c.Request.Context(), apiKeyString)
		if err != nil {
			if errors.Is(err, service.ErrAPIKeyNotFound) {
				recordInvalidAuthFailure(c, apiKeyService)
				MarkIngressRejected(c, IngressRejectInvalidAPIKey)
				abortWithGoogleError(c, 401, "Invalid API key")
				return
			}
			if errors.Is(err, service.ErrAPIKeyAuthOverloaded) {
				MarkIngressRejected(c, IngressRejectAPIKeyAuthOverloaded)
				abortWithGoogleError(c, 503, "API key authentication is temporarily unavailable")
				return
			}
			abortWithGoogleError(c, 500, "Failed to validate API key")
			return
		}

		// 同 api_key_auth.go：早退中断前也写入 Ops 回退 key，便于错误日志展示
		// user/group/platform。
		SetOpsFallbackAPIKey(c, apiKey)

		// disabled / 未知状态 → 无条件拦截（expired 和 quota_exhausted 留给计费阶段，
		// 与主中间件 api_key_auth.go 保持一致）。
		if !apiKey.IsActive() &&
			apiKey.Status != service.StatusAPIKeyExpired &&
			apiKey.Status != service.StatusAPIKeyQuotaExhausted {
			MarkIngressRejected(c, IngressRejectAPIKeyDisabled)
			abortWithGoogleError(c, 401, "API key is disabled")
			return
		}

		// 检查 IP 限制（白名单/黑名单）。与主中间件保持一致，避免 Gemini 端点绕过 Key 的 IP ACL。
		if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
			clientIP := ip.GetSecurityClientIP(c, cfg.TrustForwardedIPForAPIKeyACL())
			allowed, _ := ip.CheckIPRestrictionWithCompiledRules(clientIP, apiKey.CompiledIPWhitelist, apiKey.CompiledIPBlacklist)
			if !allowed {
				if clientIP == "" {
					clientIP = "unknown"
				}
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonIPRestriction)
				MarkIngressRejected(c, IngressRejectIPRestricted)
				abortWithGoogleError(c, 403, fmt.Sprintf("Access denied. Your IP is %s", clientIP))
				return
			}
		}

		if apiKey.User == nil {
			abortWithGoogleError(c, 401, "User associated with API key not found")
			return
		}
		if !apiKey.User.IsActive() {
			MarkIngressRejected(c, IngressRejectUserInactive)
			abortWithGoogleError(c, 401, "User account is not active")
			return
		}
		// 用户内容违规临时封禁（Redis 封禁键，鉴权缓存之外逐请求检查，到期自动恢复）
		if until, banned := apiKeyService.UserViolationBanUntil(c.Request.Context(), apiKey.User.ID); banned {
			MarkIngressRejected(c, IngressRejectUserInactive)
			abortWithGoogleError(c, 403, fmt.Sprintf("User is temporarily banned until %s due to content policy violations", until.Format("2006-01-02 15:04:05 MST")))
			return
		}
		// 分组可用性/归属检查在 selectAPIKeyGroupForRequest 选定分组后由
		// validateGoogleAPIKeyGroup 统一执行（支持 Primary/Secondary 双分组回退）。

		// 简易模式：跳过余额和订阅检查
		if cfg.RunMode == config.RunModeSimple {
			if groupErr := selectAPIKeyGroupForRequest(c.Request.Context(), apiKey, nil, true); groupErr != nil {
				abortGoogleAPIKeyGroupSelectionError(c, groupErr)
				return
			}
			if !validateGoogleAPIKeyGroup(c, apiKey) {
				return
			}
			setAPIKeyUserRequestContext(c, apiKey.User.ID)
			c.Set(string(ContextKeyAPIKey), apiKey)
			c.Set(string(ContextKeyUser), AuthSubject{
				UserID:      apiKey.User.ID,
				Concurrency: apiKey.User.Concurrency,
			})
			c.Set(string(ContextKeyUserRole), apiKey.User.Role)
			setGroupContext(c, apiKey.Group)
			_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)
			c.Next()
			return
		}

		// D2api selects a usable subscription wallet and its corresponding group
		// before applying the upstream key and limit checks.
		if groupErr := selectAPIKeyGroupForRequest(c.Request.Context(), apiKey, subscriptionService, false); groupErr != nil {
			abortGoogleAPIKeyGroupSelectionError(c, groupErr)
			return
		}
		subscription := apiKey.SelectedSubscription
		if !validateGoogleAPIKeyGroup(c, apiKey) {
			return
		}
		// Key state and runtime quota checks apply to both wallet-backed and
		// ordinary balance requests.
		switch apiKey.Status {
		case service.StatusAPIKeyQuotaExhausted:
			abortWithGoogleError(c, 429, "API key 额度已用完")
			return
		case service.StatusAPIKeyExpired:
			abortWithGoogleError(c, 403, "API key 已过期")
			return
		}
		if apiKey.IsExpired() {
			abortWithGoogleError(c, 403, "API key 已过期")
			return
		}
		if apiKey.IsQuotaExhausted() {
			abortWithGoogleError(c, 429, "API key 额度已用完")
			return
		}
		if subscription != nil {
			// Wallet exhaustion is handled atomically during billing, where the
			// remaining cost may fall back to the user's ordinary balance.
			c.Set(string(ContextKeySubscription), subscription)
		}

		setAPIKeyUserRequestContext(c, apiKey.User.ID)
		c.Set(string(ContextKeyAPIKey), apiKey)
		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), apiKey.User.Role)
		setGroupContext(c, apiKey.Group)
		_ = apiKeyService.TouchLastUsed(c.Request.Context(), apiKey.ID)
		c.Next()
	}
}

// validateGoogleAPIKeyGroup mirrors the standard middleware's group checks
// after D2api has selected the primary or fallback group for this request.
func validateGoogleAPIKeyGroup(c *gin.Context, apiKey *service.APIKey) bool {
	if code, message, ok := validateAPIKeyGroupAvailable(apiKey); !ok {
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable)
		if code == "GROUP_DELETED" {
			MarkIngressRejected(c, IngressRejectGroupDeleted)
		} else {
			MarkIngressRejected(c, IngressRejectGroupDisabled)
		}
		abortWithGoogleError(c, 403, message)
		return false
	}
	if !validateAPIKeyGroupAllowed(apiKey) {
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable)
		MarkIngressRejected(c, IngressRejectGroupNotAllowed)
		abortWithGoogleError(c, 403, "API Key 所属专属分组不再允许当前用户使用")
		return false
	}
	return true
}

// extractAPIKeyForGoogle extracts API key for Google/Gemini endpoints.
// Priority: x-goog-api-key > Authorization: Bearer > x-api-key > query key
// This allows OpenClaw and other clients using Bearer auth to work with Gemini endpoints.
func extractAPIKeyForGoogle(c *gin.Context) string {
	// 1) preferred: Gemini native header
	if k := strings.TrimSpace(c.GetHeader("x-goog-api-key")); k != "" {
		return k
	}

	// 2) fallback: Authorization: Bearer <key>
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if k := strings.TrimSpace(parts[1]); k != "" {
				return k
			}
		}
	}

	// 3) x-api-key header (backward compatibility)
	if k := strings.TrimSpace(c.GetHeader("x-api-key")); k != "" {
		return k
	}

	// 4) query parameter key (for specific paths)
	if allowGoogleQueryKey(c.Request.URL.Path) {
		if v := strings.TrimSpace(c.Query("key")); v != "" {
			return v
		}
	}

	return ""
}

func allowGoogleQueryKey(path string) bool {
	return strings.HasPrefix(path, "/v1beta") || strings.HasPrefix(path, "/antigravity/v1beta")
}

func abortWithGoogleError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleapi.HTTPStatusToGoogleStatus(status),
		},
	})
	c.Abort()
}

func abortGoogleAPIKeyGroupSelectionError(c *gin.Context, err error) {
	var selectionErr *apiKeyGroupSelectionError
	if !errors.As(err, &selectionErr) {
		abortWithGoogleError(c, 500, "Failed to select API key group")
		return
	}
	if selectionErr.markOps {
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable)
	}
	markIngressRejectedForGroupSelectionError(c, selectionErr.code)
	abortWithGoogleError(c, selectionErr.status, selectionErr.message)
}
