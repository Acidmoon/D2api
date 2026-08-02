package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ViolationBanResponse 用户内容违规临时封禁状态
type ViolationBanResponse struct {
	UserID int64      `json:"user_id"`
	Banned bool       `json:"banned"`
	Until  *time.Time `json:"until,omitempty"`
}

// SetViolationBanCache 注入用户违规封禁缓存（wire 装配时调用，可选）。
func (h *UserHandler) SetViolationBanCache(cache service.ViolationCounterCache) {
	h.violationBanCache = cache
}

// GetViolationBan 查询用户内容违规临时封禁状态
// GET /api/v1/admin/users/:id/violation-ban
func (h *UserHandler) GetViolationBan(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	result := ViolationBanResponse{UserID: userID}
	if h.violationBanCache != nil {
		until, banned, err := h.violationBanCache.GetUserViolationBan(c.Request.Context(), userID)
		if err != nil {
			response.InternalError(c, "Failed to query violation ban status")
			return
		}
		result.Banned = banned
		if banned {
			result.Until = &until
		}
	}
	response.Success(c, result)
}

// DeleteViolationBan 解除用户内容违规临时封禁
// DELETE /api/v1/admin/users/:id/violation-ban
func (h *UserHandler) DeleteViolationBan(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	if h.violationBanCache != nil {
		if err := h.violationBanCache.ClearUserViolationBan(c.Request.Context(), userID); err != nil {
			response.InternalError(c, "Failed to clear violation ban")
			return
		}
	}
	response.Success(c, ViolationBanResponse{UserID: userID, Banned: false})
}
