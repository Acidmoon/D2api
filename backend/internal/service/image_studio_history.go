package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"go.uber.org/zap"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 生图创作中心历史的保留与限额。图片字节落在 <data_dir>/<user_id>/<随机目录>/<index>.<ext>，
// 元数据落在 image_studio_history 表；user_id 是唯一的可见性边界。
const (
	defaultImageStudioRetentionDays  = 7
	defaultImageStudioMaxPerUser     = 200
	defaultImageStudioMaxImageBytes  = 12 << 20
	defaultImageStudioCleanupMinutes = 60
	imageStudioMaxImagesPerRecord    = 8
	imageStudioMaxPromptRunes        = 4000
	imageStudioCleanupBatchSize      = 200
	imageStudioDefaultDataDir        = "./data/image-studio"
	imageStudioHistoryFileMode       = 0o640
	imageStudioHistoryDirMode        = 0o750
)

var (
	// ErrImageStudioHistoryNotFound 记录不存在或不属于该用户（对外统一返回，避免探测他人记录）。
	ErrImageStudioHistoryNotFound = errors.New("image studio history not found")
	// ErrImageStudioHistoryInvalid 请求内容不合法（无有效图片、缺少提示词/模型等）。
	ErrImageStudioHistoryInvalid = errors.New("invalid image studio history payload")
)

// ImageStudioHistoryImage 一条记录中单张图片的落盘信息（File 为相对 data_dir 的路径）。
type ImageStudioHistoryImage struct {
	File  string `json:"file"`
	Mime  string `json:"mime"`
	Bytes int64  `json:"bytes"`
}

// ImageStudioHistoryRecord 一次生成的完整记录。
type ImageStudioHistoryRecord struct {
	ID            int64
	UserID        int64
	APIKeyID      *int64
	GroupID       *int64
	Model         string
	Prompt        string
	RevisedPrompt string
	Size          string
	Images        []ImageStudioHistoryImage
	ImageCount    int
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

// ImageStudioHistoryImageInput 前端提交的单张图片（Base64，可含 data URL 前缀）。
type ImageStudioHistoryImageInput struct {
	MimeType string
	Base64   string
}

// ImageStudioHistorySaveInput 保存一次生成的入参。
type ImageStudioHistorySaveInput struct {
	UserID        int64
	APIKeyID      *int64
	GroupID       *int64
	Model         string
	Prompt        string
	RevisedPrompt string
	Size          string
	Images        []ImageStudioHistoryImageInput
}

// ImageStudioHistoryRepository 历史记录的持久化接口。
type ImageStudioHistoryRepository interface {
	Create(ctx context.Context, rec *ImageStudioHistoryRecord) error
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]ImageStudioHistoryRecord, int, error)
	GetByID(ctx context.Context, userID, id int64) (*ImageStudioHistoryRecord, error)
	DeleteByID(ctx context.Context, userID, id int64) (*ImageStudioHistoryRecord, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	ListOldestByUser(ctx context.Context, userID int64, limit int) ([]ImageStudioHistoryRecord, error)
	ListExpired(ctx context.Context, before time.Time, limit int) ([]ImageStudioHistoryRecord, error)
	DeleteByIDs(ctx context.Context, ids []int64) error
}

// ImageStudioHistoryService 负责历史的落盘、查询与过期清理。
//
// 设计要点：
//   - 只保存创作中心主动提交的结果（前端生成成功后调用保存接口），不改动网关热路径；
//   - 过期时间在写入时确定（created_at + retention_days），清理 worker 定时物理删除；
//   - 单用户条数超限时删除最旧记录，避免磁盘被单个用户撑满。
type ImageStudioHistoryService struct {
	repo ImageStudioHistoryRepository
	cfg  *config.Config

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewImageStudioHistoryService 构造历史服务（不启动清理 worker，由 Provide 阶段调用 Start）。
func NewImageStudioHistoryService(repo ImageStudioHistoryRepository, cfg *config.Config) *ImageStudioHistoryService {
	return &ImageStudioHistoryService{repo: repo, cfg: cfg}
}

func (s *ImageStudioHistoryService) dataDir() string {
	dir := ""
	if s != nil && s.cfg != nil {
		dir = strings.TrimSpace(s.cfg.ImageStudio.DataDir)
	}
	if dir == "" {
		dir = imageStudioDefaultDataDir
	}
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

func (s *ImageStudioHistoryService) retentionDays() int {
	if s != nil && s.cfg != nil && s.cfg.ImageStudio.RetentionDays > 0 {
		return s.cfg.ImageStudio.RetentionDays
	}
	return defaultImageStudioRetentionDays
}

func (s *ImageStudioHistoryService) maxPerUser() int {
	if s != nil && s.cfg != nil && s.cfg.ImageStudio.MaxPerUser > 0 {
		return s.cfg.ImageStudio.MaxPerUser
	}
	return defaultImageStudioMaxPerUser
}

func (s *ImageStudioHistoryService) maxImageBytes() int64 {
	if s != nil && s.cfg != nil && s.cfg.ImageStudio.MaxImageBytes > 0 {
		return s.cfg.ImageStudio.MaxImageBytes
	}
	return defaultImageStudioMaxImageBytes
}

func (s *ImageStudioHistoryService) cleanupInterval() time.Duration {
	minutes := defaultImageStudioCleanupMinutes
	if s != nil && s.cfg != nil && s.cfg.ImageStudio.CleanupIntervalMinutes > 0 {
		minutes = s.cfg.ImageStudio.CleanupIntervalMinutes
	}
	return time.Duration(minutes) * time.Minute
}

// Save 落盘并写入元数据；无有效图片时返回 ErrImageStudioHistoryInvalid。
func (s *ImageStudioHistoryService) Save(ctx context.Context, in ImageStudioHistorySaveInput) (*ImageStudioHistoryRecord, error) {
	if s == nil || s.repo == nil {
		return nil, ErrImageStudioHistoryInvalid
	}
	in.Model = strings.TrimSpace(in.Model)
	in.Prompt = strings.TrimSpace(in.Prompt)
	if in.UserID <= 0 || in.Model == "" || in.Prompt == "" || len(in.Images) == 0 {
		return nil, ErrImageStudioHistoryInvalid
	}
	if utf8.RuneCountInString(in.Prompt) > imageStudioMaxPromptRunes {
		in.Prompt = string([]rune(in.Prompt)[:imageStudioMaxPromptRunes])
	}
	if len(in.Images) > imageStudioMaxImagesPerRecord {
		in.Images = in.Images[:imageStudioMaxImagesPerRecord]
	}

	dirName, err := randomImageStudioDirName()
	if err != nil {
		return nil, err
	}
	relDir := filepath.Join(strconv.FormatInt(in.UserID, 10), dirName)
	absDir := filepath.Join(s.dataDir(), relDir)
	if err := os.MkdirAll(absDir, imageStudioHistoryDirMode); err != nil {
		return nil, fmt.Errorf("create image studio dir: %w", err)
	}

	images := make([]ImageStudioHistoryImage, 0, len(in.Images))
	for _, image := range in.Images {
		raw, mimeType, decodeErr := decodeImageStudioPayload(image)
		if decodeErr != nil {
			continue
		}
		if int64(len(raw)) > s.maxImageBytes() {
			logger.L().Warn("image_studio.history_image_skipped_too_large",
				zap.Int("bytes", len(raw)), zap.Int64("max_bytes", s.maxImageBytes()))
			continue
		}
		name := fmt.Sprintf("%d.%s", len(images), imageStudioExtensionForMime(mimeType))
		abs := filepath.Join(absDir, name)
		if writeErr := os.WriteFile(abs, raw, imageStudioHistoryFileMode); writeErr != nil {
			logger.L().Warn("image_studio.history_image_write_failed", zap.Error(writeErr))
			continue
		}
		images = append(images, ImageStudioHistoryImage{
			File:  filepath.Join(relDir, name),
			Mime:  mimeType,
			Bytes: int64(len(raw)),
		})
	}
	if len(images) == 0 {
		_ = os.RemoveAll(absDir)
		return nil, ErrImageStudioHistoryInvalid
	}

	now := time.Now()
	rec := &ImageStudioHistoryRecord{
		UserID:        in.UserID,
		APIKeyID:      in.APIKeyID,
		GroupID:       in.GroupID,
		Model:         in.Model,
		Prompt:        in.Prompt,
		RevisedPrompt: strings.TrimSpace(in.RevisedPrompt),
		Size:          strings.TrimSpace(in.Size),
		Images:        images,
		ImageCount:    len(images),
		CreatedAt:     now,
		ExpiresAt:     now.AddDate(0, 0, s.retentionDays()),
	}
	if err := s.repo.Create(ctx, rec); err != nil {
		_ = os.RemoveAll(absDir)
		return nil, err
	}
	s.enforceUserCap(ctx, in.UserID)
	return rec, nil
}

// List 返回该用户未过期的历史（created_at 倒序）。
func (s *ImageStudioHistoryService) List(ctx context.Context, userID int64, page, pageSize int) ([]ImageStudioHistoryRecord, int, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, 0, ErrImageStudioHistoryInvalid
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListByUser(ctx, userID, pageSize, (page-1)*pageSize)
}

// Get 返回单条记录（含图片索引信息），非本人记录统一返回 NotFound。
func (s *ImageStudioHistoryService) Get(ctx context.Context, userID, id int64) (*ImageStudioHistoryRecord, error) {
	if s == nil || s.repo == nil || userID <= 0 || id <= 0 {
		return nil, ErrImageStudioHistoryNotFound
	}
	return s.repo.GetByID(ctx, userID, id)
}

// OpenImage 打开某张图片，返回内容与 MIME；调用方负责 Close。
func (s *ImageStudioHistoryService) OpenImage(ctx context.Context, userID, id int64, index int) (io.ReadCloser, string, error) {
	rec, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, "", err
	}
	if index < 0 || index >= len(rec.Images) {
		return nil, "", ErrImageStudioHistoryNotFound
	}
	image := rec.Images[index]
	abs := filepath.Join(s.dataDir(), filepath.Clean(image.File))
	file, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", ErrImageStudioHistoryNotFound
		}
		return nil, "", err
	}
	mimeType := image.Mime
	if mimeType == "" {
		mimeType = "image/png"
	}
	return file, mimeType, nil
}

// Delete 删除单条记录及其文件。
func (s *ImageStudioHistoryService) Delete(ctx context.Context, userID, id int64) error {
	if s == nil || s.repo == nil || userID <= 0 || id <= 0 {
		return ErrImageStudioHistoryNotFound
	}
	rec, err := s.repo.DeleteByID(ctx, userID, id)
	if err != nil {
		return err
	}
	s.removeRecordFiles(rec)
	return nil
}

func (s *ImageStudioHistoryService) enforceUserCap(ctx context.Context, userID int64) {
	cap := s.maxPerUser()
	if cap <= 0 {
		return
	}
	count, err := s.repo.CountByUser(ctx, userID)
	if err != nil || count <= cap {
		return
	}
	oldest, err := s.repo.ListOldestByUser(ctx, userID, count-cap)
	if err != nil || len(oldest) == 0 {
		return
	}
	ids := make([]int64, 0, len(oldest))
	for i := range oldest {
		s.removeRecordFiles(&oldest[i])
		ids = append(ids, oldest[i].ID)
	}
	if err := s.repo.DeleteByIDs(ctx, ids); err != nil {
		logger.L().Warn("image_studio.history_cap_delete_failed", zap.Error(err))
	}
}

func (s *ImageStudioHistoryService) removeRecordFiles(rec *ImageStudioHistoryRecord) {
	if rec == nil || len(rec.Images) == 0 {
		return
	}
	dirs := make(map[string]struct{}, 1)
	for _, image := range rec.Images {
		abs := filepath.Join(s.dataDir(), filepath.Clean(image.File))
		_ = os.Remove(abs)
		dirs[filepath.Dir(abs)] = struct{}{}
	}
	for dir := range dirs {
		_ = os.Remove(dir)
	}
}

// RunOnce 删除一批过期记录（含文件），返回删除条数。
func (s *ImageStudioHistoryService) RunOnce(ctx context.Context, now time.Time) (int, error) {
	if s == nil || s.repo == nil {
		return 0, ErrImageStudioHistoryInvalid
	}
	if now.IsZero() {
		now = time.Now()
	}
	expired, err := s.repo.ListExpired(ctx, now, imageStudioCleanupBatchSize)
	if err != nil || len(expired) == 0 {
		return 0, err
	}
	ids := make([]int64, 0, len(expired))
	for i := range expired {
		s.removeRecordFiles(&expired[i])
		ids = append(ids, expired[i].ID)
	}
	if err := s.repo.DeleteByIDs(ctx, ids); err != nil {
		return 0, err
	}
	return len(ids), nil
}

// Start 启动后台清理 worker（幂等）。
func (s *ImageStudioHistoryService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	interval := s.cleanupInterval()
	if interval <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		if _, err := s.RunOnce(ctx, time.Now()); err != nil {
			logger.L().Warn("image_studio.history_cleanup_failed", zap.Error(err))
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := s.RunOnce(ctx, time.Now()); err != nil {
					logger.L().Warn("image_studio.history_cleanup_failed", zap.Error(err))
				}
			}
		}
	}()
}

// Stop 停止清理 worker。
func (s *ImageStudioHistoryService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.cancel = nil
	s.done = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func randomImageStudioDirName() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// decodeImageStudioPayload 解码 Base64（容忍 data URL 前缀），并按魔数纠正 MIME。
func decodeImageStudioPayload(in ImageStudioHistoryImageInput) ([]byte, string, error) {
	data := strings.TrimSpace(in.Base64)
	if data == "" {
		return nil, "", errors.New("empty image data")
	}
	if idx := strings.Index(data, ","); idx > 0 && strings.HasPrefix(data, "data:") {
		data = data[idx+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", err
	}
	if len(raw) == 0 {
		return nil, "", errors.New("empty image bytes")
	}
	mimeType := strings.TrimSpace(in.MimeType)
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = detectImageStudioMime(raw)
	}
	return raw, mimeType, nil
}

func detectImageStudioMime(raw []byte) string {
	switch {
	case len(raw) >= 8 && string(raw[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(raw) >= 3 && raw[0] == 0xff && raw[1] == 0xd8 && raw[2] == 0xff:
		return "image/jpeg"
	case len(raw) >= 12 && string(raw[:4]) == "RIFF" && string(raw[8:12]) == "WEBP":
		return "image/webp"
	case len(raw) >= 6 && (string(raw[:6]) == "GIF87a" || string(raw[:6]) == "GIF89a"):
		return "image/gif"
	default:
		return "image/png"
	}
}

func imageStudioExtensionForMime(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "png"
	}
}
