package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 单次保存请求体上限：8 张 12MiB 图片的 base64 展开后仍有余量。
const imageStudioHistoryMaxBodyBytes = 64 << 20

// ImageStudioHandler 生图创作中心的历史记录接口（用户会话鉴权，按 user_id 隔离）。
type ImageStudioHandler struct {
	service *service.ImageStudioHistoryService
}

// NewImageStudioHandler 构造历史记录 handler。
func NewImageStudioHandler(historyService *service.ImageStudioHistoryService) *ImageStudioHandler {
	return &ImageStudioHandler{service: historyService}
}

type imageStudioHistoryImageRequest struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type imageStudioHistorySaveRequest struct {
	Model         string                           `json:"model" binding:"required"`
	Prompt        string                           `json:"prompt" binding:"required"`
	RevisedPrompt string                           `json:"revised_prompt"`
	Size          string                           `json:"size"`
	APIKeyID      *int64                           `json:"api_key_id"`
	GroupID       *int64                           `json:"group_id"`
	Images        []imageStudioHistoryImageRequest `json:"images" binding:"required"`
}

type imageStudioHistoryImageView struct {
	Index    int    `json:"index"`
	MimeType string `json:"mime_type"`
	Bytes    int64  `json:"bytes"`
}

type imageStudioHistoryView struct {
	ID            int64                         `json:"id"`
	Model         string                        `json:"model"`
	Prompt        string                        `json:"prompt"`
	RevisedPrompt string                        `json:"revised_prompt"`
	Size          string                        `json:"size"`
	ImageCount    int                           `json:"image_count"`
	CreatedAt     string                        `json:"created_at"`
	ExpiresAt     string                        `json:"expires_at"`
	Images        []imageStudioHistoryImageView `json:"images"`
}

func imageStudioHistoryToView(rec *service.ImageStudioHistoryRecord) imageStudioHistoryView {
	images := make([]imageStudioHistoryImageView, 0, len(rec.Images))
	for i, image := range rec.Images {
		images = append(images, imageStudioHistoryImageView{
			Index:    i,
			MimeType: image.Mime,
			Bytes:    image.Bytes,
		})
	}
	return imageStudioHistoryView{
		ID:            rec.ID,
		Model:         rec.Model,
		Prompt:        rec.Prompt,
		RevisedPrompt: rec.RevisedPrompt,
		Size:          rec.Size,
		ImageCount:    rec.ImageCount,
		CreatedAt:     rec.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		ExpiresAt:     rec.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		Images:        images,
	}
}

// Save 保存一次生成
// POST /api/v1/image-studio/history
func (h *ImageStudioHandler) Save(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, imageStudioHistoryMaxBodyBytes)

	var req imageStudioHistorySaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	images := make([]service.ImageStudioHistoryImageInput, 0, len(req.Images))
	for _, image := range req.Images {
		images = append(images, service.ImageStudioHistoryImageInput{
			MimeType: image.MimeType,
			Base64:   image.Data,
		})
	}

	rec, err := h.service.Save(c.Request.Context(), service.ImageStudioHistorySaveInput{
		UserID:        subject.UserID,
		APIKeyID:      req.APIKeyID,
		GroupID:       req.GroupID,
		Model:         req.Model,
		Prompt:        req.Prompt,
		RevisedPrompt: req.RevisedPrompt,
		Size:          req.Size,
		Images:        images,
	})
	if err != nil {
		if err == service.ErrImageStudioHistoryInvalid {
			response.BadRequest(c, "No valid image to save")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, imageStudioHistoryToView(rec))
}

// List 列出当前用户的历史
// GET /api/v1/image-studio/history?page=1&page_size=20
func (h *ImageStudioHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	records, total, err := h.service.List(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]imageStudioHistoryView, 0, len(records))
	for i := range records {
		items = append(items, imageStudioHistoryToView(&records[i]))
	}
	response.Success(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 查看单条历史
// GET /api/v1/image-studio/history/:id
func (h *ImageStudioHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid history id")
		return
	}
	rec, err := h.service.Get(c.Request.Context(), subject.UserID, id)
	if err != nil {
		if err == service.ErrImageStudioHistoryNotFound {
			response.Error(c, http.StatusNotFound, "History not found")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, imageStudioHistoryToView(rec))
}

// Image 读取历史中的原图（仅本人）
// GET /api/v1/image-studio/history/:id/images/:index
func (h *ImageStudioHandler) Image(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid history id")
		return
	}
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 0 {
		response.BadRequest(c, "Invalid image index")
		return
	}

	reader, mimeType, err := h.service.OpenImage(c.Request.Context(), subject.UserID, id, index)
	if err != nil {
		if err == service.ErrImageStudioHistoryNotFound {
			response.Error(c, http.StatusNotFound, "Image not found")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	defer func() { _ = reader.Close() }()

	if mimeType == "" {
		mimeType = "image/png"
	}
	// 历史图片属于私人内容：仅允许浏览器私有缓存，不落共享缓存。
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Type", mimeType)
	if name := c.Query("download"); strings.TrimSpace(name) != "" {
		c.Header("Content-Disposition", "attachment; filename=\""+sanitizeDownloadFilename(name)+"\"")
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, reader)
}

// Delete 删除单条历史
// DELETE /api/v1/image-studio/history/:id
func (h *ImageStudioHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid history id")
		return
	}
	if err := h.service.Delete(c.Request.Context(), subject.UserID, id); err != nil {
		if err == service.ErrImageStudioHistoryNotFound {
			response.Error(c, http.StatusNotFound, "History not found")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

func sanitizeDownloadFilename(name string) string {
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_' || r == ' ':
			return r
		default:
			return -1
		}
	}, name)
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "image.png"
	}
	return clean
}
