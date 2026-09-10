//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// stubImageStudioHistoryRepo 用内存切片模拟持久层，便于在无数据库环境下测试服务层行为。
type stubImageStudioHistoryRepo struct {
	mu      sync.Mutex
	seq     int64
	records []ImageStudioHistoryRecord
}

func (r *stubImageStudioHistoryRepo) Create(_ context.Context, rec *ImageStudioHistoryRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	rec.ID = r.seq
	r.records = append(r.records, *rec)
	return nil
}

func (r *stubImageStudioHistoryRepo) ListByUser(_ context.Context, userID int64, limit, offset int) ([]ImageStudioHistoryRecord, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	mine := r.mineLocked(userID)
	sort.Slice(mine, func(i, j int) bool { return mine[i].CreatedAt.After(mine[j].CreatedAt) })
	total := len(mine)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	out := make([]ImageStudioHistoryRecord, end-offset)
	copy(out, mine[offset:end])
	return out, total, nil
}

func (r *stubImageStudioHistoryRepo) GetByID(_ context.Context, userID, id int64) (*ImageStudioHistoryRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.records {
		if r.records[i].ID == id && r.records[i].UserID == userID {
			rec := r.records[i]
			return &rec, nil
		}
	}
	return nil, ErrImageStudioHistoryNotFound
}

func (r *stubImageStudioHistoryRepo) DeleteByID(_ context.Context, userID, id int64) (*ImageStudioHistoryRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.records {
		if r.records[i].ID == id && r.records[i].UserID == userID {
			rec := r.records[i]
			r.records = append(r.records[:i], r.records[i+1:]...)
			return &rec, nil
		}
	}
	return nil, ErrImageStudioHistoryNotFound
}

func (r *stubImageStudioHistoryRepo) CountByUser(_ context.Context, userID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.mineLocked(userID)), nil
}

func (r *stubImageStudioHistoryRepo) ListOldestByUser(_ context.Context, userID int64, limit int) ([]ImageStudioHistoryRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	mine := r.mineLocked(userID)
	sort.Slice(mine, func(i, j int) bool { return mine[i].CreatedAt.Before(mine[j].CreatedAt) })
	if limit > len(mine) {
		limit = len(mine)
	}
	out := make([]ImageStudioHistoryRecord, limit)
	copy(out, mine[:limit])
	return out, nil
}

func (r *stubImageStudioHistoryRepo) ListExpired(_ context.Context, before time.Time, limit int) ([]ImageStudioHistoryRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ImageStudioHistoryRecord, 0, limit)
	for i := range r.records {
		if r.records[i].ExpiresAt.After(before) {
			continue
		}
		out = append(out, r.records[i])
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *stubImageStudioHistoryRepo) DeleteByIDs(_ context.Context, ids []int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	drop := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		drop[id] = struct{}{}
	}
	kept := r.records[:0]
	for i := range r.records {
		if _, ok := drop[r.records[i].ID]; ok {
			continue
		}
		kept = append(kept, r.records[i])
	}
	r.records = kept
	return nil
}

func (r *stubImageStudioHistoryRepo) mineLocked(userID int64) []ImageStudioHistoryRecord {
	out := make([]ImageStudioHistoryRecord, 0, len(r.records))
	for i := range r.records {
		if r.records[i].UserID == userID {
			out = append(out, r.records[i])
		}
	}
	return out
}

func (r *stubImageStudioHistoryRepo) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.records)
}

func newImageStudioHistoryTestService(t *testing.T, repo ImageStudioHistoryRepository, mutate func(cfg *config.ImageStudioConfig)) *ImageStudioHistoryService {
	t.Helper()
	studioCfg := config.ImageStudioConfig{
		DataDir:       t.TempDir(),
		RetentionDays: 7,
		MaxPerUser:    10,
		MaxImageBytes: 1 << 20,
	}
	if mutate != nil {
		mutate(&studioCfg)
	}
	return NewImageStudioHistoryService(repo, &config.Config{ImageStudio: studioCfg})
}

// minimalPNG 返回一段带 PNG 魔数的字节，用于验证 MIME 嗅探与落盘。
func minimalPNG(size int) []byte {
	header := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	if size <= len(header) {
		return header
	}
	raw := make([]byte, size)
	copy(raw, header)
	return raw
}

func encodeImage(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func TestImageStudioHistorySaveWritesFilesAndMetadata(t *testing.T) {
	repo := &stubImageStudioHistoryRepo{}
	svc := newImageStudioHistoryTestService(t, repo, nil)

	rec, err := svc.Save(context.Background(), ImageStudioHistorySaveInput{
		UserID: 7,
		Model:  "gpt-image-2.5",
		Prompt: "一只戴宇航头盔的橘猫",
		Size:   "1024x1024",
		Images: []ImageStudioHistoryImageInput{{Base64: encodeImage(minimalPNG(64))}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, rec.ImageCount)
	require.Len(t, rec.Images, 1)
	require.Equal(t, "image/png", rec.Images[0].Mime)
	require.Equal(t, "png", filepath.Ext(rec.Images[0].File)[1:])

	abs := filepath.Join(svc.dataDir(), rec.Images[0].File)
	_, statErr := os.Stat(abs)
	require.NoError(t, statErr)

	// 过期时间应为 7 天后（允许分钟级误差）。
	expectedExpiry := rec.CreatedAt.AddDate(0, 0, 7)
	require.WithinDuration(t, expectedExpiry, rec.ExpiresAt, time.Minute)
}

func TestImageStudioHistoryRejectsPayloadWithoutUsableImage(t *testing.T) {
	repo := &stubImageStudioHistoryRepo{}
	svc := newImageStudioHistoryTestService(t, repo, func(cfg *config.ImageStudioConfig) {
		cfg.MaxImageBytes = 4 // 让 64 字节的图片超限被跳过
	})

	_, err := svc.Save(context.Background(), ImageStudioHistorySaveInput{
		UserID: 7,
		Model:  "gpt-image-2.5",
		Prompt: "超限图片",
		Images: []ImageStudioHistoryImageInput{{Base64: encodeImage(minimalPNG(64))}},
	})
	require.ErrorIs(t, err, ErrImageStudioHistoryInvalid)
	require.Equal(t, 0, repo.total())

	// 缺少提示词同样拒绝。
	_, err = svc.Save(context.Background(), ImageStudioHistorySaveInput{
		UserID: 7,
		Model:  "gpt-image-2.5",
		Images: []ImageStudioHistoryImageInput{{Base64: encodeImage(minimalPNG(16))}},
	})
	require.ErrorIs(t, err, ErrImageStudioHistoryInvalid)
}

func TestImageStudioHistoryIsPerUser(t *testing.T) {
	repo := &stubImageStudioHistoryRepo{}
	svc := newImageStudioHistoryTestService(t, repo, nil)

	rec, err := svc.Save(context.Background(), ImageStudioHistorySaveInput{
		UserID: 7,
		Model:  "gpt-image-2.5",
		Prompt: "owner 7",
		Images: []ImageStudioHistoryImageInput{{Base64: encodeImage(minimalPNG(32))}},
	})
	require.NoError(t, err)

	// 本人可见
	_, mimeType, err := svc.OpenImage(context.Background(), 7, rec.ID, 0)
	require.NoError(t, err)
	require.Equal(t, "image/png", mimeType)

	// 他人不可见（列表、详情、图片一律 NotFound）
	otherList, total, err := svc.List(context.Background(), 8, 1, 20)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, otherList)

	_, err = svc.Get(context.Background(), 8, rec.ID)
	require.ErrorIs(t, err, ErrImageStudioHistoryNotFound)

	_, _, err = svc.OpenImage(context.Background(), 8, rec.ID, 0)
	require.ErrorIs(t, err, ErrImageStudioHistoryNotFound)
}

func TestImageStudioHistoryCleanupRemovesExpiredRecordsAndFiles(t *testing.T) {
	repo := &stubImageStudioHistoryRepo{}
	svc := newImageStudioHistoryTestService(t, repo, nil)

	rec, err := svc.Save(context.Background(), ImageStudioHistorySaveInput{
		UserID: 7,
		Model:  "gpt-image-2.5",
		Prompt: "expired",
		Images: []ImageStudioHistoryImageInput{{Base64: encodeImage(minimalPNG(32))}},
	})
	require.NoError(t, err)
	abs := filepath.Join(svc.dataDir(), rec.Images[0].File)

	// 手动把过期时间提前，模拟 7 天后的清理。
	repo.mu.Lock()
	repo.records[0].ExpiresAt = time.Now().Add(-time.Minute)
	repo.mu.Unlock()

	removed, err := svc.RunOnce(context.Background(), time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	require.Equal(t, 0, repo.total())
	_, statErr := os.Stat(abs)
	require.True(t, os.IsNotExist(statErr), "过期图片文件应被删除")
}

func TestImageStudioHistoryCapDropsOldestRecord(t *testing.T) {
	repo := &stubImageStudioHistoryRepo{}
	svc := newImageStudioHistoryTestService(t, repo, func(cfg *config.ImageStudioConfig) {
		cfg.MaxPerUser = 2
	})

	var last *ImageStudioHistoryRecord
	for i := 0; i < 3; i++ {
		rec, err := svc.Save(context.Background(), ImageStudioHistorySaveInput{
			UserID: 7,
			Model:  "gpt-image-2.5",
			Prompt: "prompt",
			Images: []ImageStudioHistoryImageInput{{Base64: encodeImage(minimalPNG(32))}},
		})
		require.NoError(t, err)
		last = rec
		time.Sleep(2 * time.Millisecond) // 保证 created_at 单调，便于判断最旧记录
	}

	records, total, err := svc.List(context.Background(), 7, 1, 20)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, records, 2)

	// 最新一条仍在
	_, err = svc.Get(context.Background(), 7, last.ID)
	require.NoError(t, err)
}
