package admin

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// FingerprintHandler 模型指纹检测管理后台 handler（全部仅 admin）。
type FingerprintHandler struct {
	fingerprintService *service.FingerprintService
}

// NewFingerprintHandler 创建 handler。
func NewFingerprintHandler(fingerprintService *service.FingerprintService) *FingerprintHandler {
	return &FingerprintHandler{fingerprintService: fingerprintService}
}

// --- Request ---

type fingerprintAuditCreateRequest struct {
	TargetType string `json:"target_type" binding:"required,oneof=account external"`
	AccountID  *int64 `json:"account_id"`
	// 以下三个仅 target_type=external 使用；api_key 只在任务运行期持有，不落盘。
	BaseURL  string `json:"base_url" binding:"omitempty,max=500"`
	APIKey   string `json:"api_key" binding:"omitempty,max=2000"`
	Provider string `json:"provider" binding:"omitempty,oneof=openai anthropic gemini grok"`
	APIMode  string `json:"api_mode" binding:"omitempty,oneof=chat_completions responses"`

	Model          string `json:"model" binding:"required,max=200"`
	ReferenceModel string `json:"reference_model" binding:"required,max=200"`
	// ReferenceAccountID 提供时先对该账号现场采样注册参考，再测目标。
	ReferenceAccountID *int64 `json:"reference_account_id"`
	KeepRaw            bool   `json:"keep_raw"`
	// 执行节奏（可选）：并发 0=默认 2（clamp 1–16），间隔 nil=默认 500ms（clamp 0–60000）。
	Concurrency int  `json:"concurrency"`
	IntervalMs  *int `json:"interval_ms"`
}

type fingerprintReferenceCreateRequest struct {
	AccountID int64  `json:"account_id" binding:"required"`
	Model     string `json:"model" binding:"required,max=200"`
	// 执行节奏（可选）：同 audits。
	Concurrency int  `json:"concurrency"`
	IntervalMs  *int `json:"interval_ms"`
}

// --- Handlers ---

// CreateAudit POST /api/v1/admin/fingerprint/audits
// 发起一次指纹检测：异步执行，返回任务状态（task_id），前端轮询 GET /audits/:id。
func (h *FingerprintHandler) CreateAudit(c *gin.Context) {
	var req fingerprintAuditCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}

	subject, _ := middleware2.GetAuthSubjectFromContext(c)

	params := service.FingerprintAuditParams{
		TargetType:     req.TargetType,
		BaseURL:        req.BaseURL,
		APIKey:         req.APIKey,
		Provider:       req.Provider,
		APIMode:        req.APIMode,
		Model:          req.Model,
		ReferenceModel: req.ReferenceModel,
		KeepRaw:        req.KeepRaw,
		Concurrency:    req.Concurrency,
		IntervalMs:     req.IntervalMs,
		OperatorID:     subject.UserID,
	}
	if req.AccountID != nil {
		params.AccountID = *req.AccountID
	}
	if req.ReferenceAccountID != nil {
		params.ReferenceAccountID = *req.ReferenceAccountID
	}

	status, err := h.fingerprintService.StartAudit(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, status)
}

// ListAudits GET /api/v1/admin/fingerprint/audits
// 检测记录列表（扫 audits 目录按时间倒序 + 内存中尚未落盘的任务）。
func (h *FingerprintHandler) ListAudits(c *gin.Context) {
	items, err := h.fingerprintService.ListAudits()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

// GetAudit GET /api/v1/admin/fingerprint/audits/:id
// 进行中返回内存任务状态（包装成 {task: ...}）；完成/失败返回报告文件内容。
func (h *FingerprintHandler) GetAudit(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_AUDIT_ID", "invalid audit id"))
		return
	}
	status, report, err := h.fingerprintService.GetAudit(id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if report != nil {
		response.Success(c, report)
		return
	}
	response.Success(c, status)
}

// RegisterReference POST /api/v1/admin/fingerprint/references
// 对可信账号现场采样注册参考指纹：异步执行，返回任务状态（task_id）。
func (h *FingerprintHandler) RegisterReference(c *gin.Context) {
	var req fingerprintReferenceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}

	status, err := h.fingerprintService.StartReferenceRegistration(c.Request.Context(), req.AccountID, req.Model, req.Concurrency, req.IntervalMs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, status)
}

// ListReferences GET /api/v1/admin/fingerprint/references
// 参考指纹列表（扫 references 目录，按注册时间倒序）。
func (h *FingerprintHandler) ListReferences(c *gin.Context) {
	items, err := h.fingerprintService.ListReferences()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

// DeleteReference DELETE /api/v1/admin/fingerprint/references/:model
// 删除参考指纹文件（model 参数经 slug 化，与写文件同一规则）。
func (h *FingerprintHandler) DeleteReference(c *gin.Context) {
	model := strings.TrimSpace(c.Param("model"))
	if model == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_MODEL", "invalid model"))
		return
	}
	if err := h.fingerprintService.DeleteReference(model); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// DeleteAudit DELETE /api/v1/admin/fingerprint/audits/:id
// 删除检测报告文件；running 中的任务拒绝（409）。
func (h *FingerprintHandler) DeleteAudit(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_AUDIT_ID", "invalid audit id"))
		return
	}
	if err := h.fingerprintService.DeleteAudit(id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
