package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// imageStudioHistorySQL 在通用 sqlExecutor 之上额外需要单行查询。
type imageStudioHistorySQL interface {
	sqlExecutor
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// imageStudioHistoryRepository 用原生 SQL 实现生图历史表（元数据部分）。
// 图片字节由 service 层负责落盘，这里只存相对路径与 MIME。
type imageStudioHistoryRepository struct {
	sql imageStudioHistorySQL
}

// NewImageStudioHistoryRepository 构造生图历史仓库。
func NewImageStudioHistoryRepository(sqlDB *sql.DB) service.ImageStudioHistoryRepository {
	return &imageStudioHistoryRepository{sql: sqlDB}
}

const imageStudioHistoryColumns = `id, user_id, api_key_id, group_id, model, prompt,
	COALESCE(revised_prompt, ''), COALESCE(size, ''), images, image_count, created_at, expires_at`

func (r *imageStudioHistoryRepository) Create(ctx context.Context, rec *service.ImageStudioHistoryRecord) error {
	if rec == nil {
		return service.ErrImageStudioHistoryInvalid
	}
	payload, err := json.Marshal(rec.Images)
	if err != nil {
		return err
	}
	return r.sql.QueryRowContext(ctx, `
		INSERT INTO image_studio_history (
			user_id, api_key_id, group_id, model, prompt, revised_prompt, size,
			images, image_count, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		rec.UserID, rec.APIKeyID, rec.GroupID, rec.Model, rec.Prompt, nullableString(rec.RevisedPrompt),
		nullableString(rec.Size), payload, rec.ImageCount, rec.CreatedAt, rec.ExpiresAt,
	).Scan(&rec.ID)
}

func (r *imageStudioHistoryRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]service.ImageStudioHistoryRecord, int, error) {
	var total int
	if err := r.sql.QueryRowContext(ctx,
		`SELECT count(*) FROM image_studio_history WHERE user_id = $1 AND expires_at > now()`,
		userID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.sql.QueryContext(ctx, `
		SELECT `+imageStudioHistoryColumns+`
		FROM image_studio_history
		WHERE user_id = $1 AND expires_at > now()
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	records := make([]service.ImageStudioHistoryRecord, 0, limit)
	for rows.Next() {
		rec, scanErr := scanImageStudioHistory(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, *rec)
	}
	return records, total, rows.Err()
}

func (r *imageStudioHistoryRepository) GetByID(ctx context.Context, userID, id int64) (*service.ImageStudioHistoryRecord, error) {
	row := r.sql.QueryRowContext(ctx, `
		SELECT `+imageStudioHistoryColumns+`
		FROM image_studio_history
		WHERE id = $1 AND user_id = $2 AND expires_at > now()`, id, userID)
	rec, err := scanImageStudioHistory(row)
	if err == sql.ErrNoRows {
		return nil, service.ErrImageStudioHistoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *imageStudioHistoryRepository) DeleteByID(ctx context.Context, userID, id int64) (*service.ImageStudioHistoryRecord, error) {
	row := r.sql.QueryRowContext(ctx, `
		DELETE FROM image_studio_history
		WHERE id = $1 AND user_id = $2
		RETURNING `+imageStudioHistoryColumns, id, userID)
	rec, err := scanImageStudioHistory(row)
	if err == sql.ErrNoRows {
		return nil, service.ErrImageStudioHistoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *imageStudioHistoryRepository) CountByUser(ctx context.Context, userID int64) (int, error) {
	var total int
	err := r.sql.QueryRowContext(ctx,
		`SELECT count(*) FROM image_studio_history WHERE user_id = $1`, userID).Scan(&total)
	return total, err
}

func (r *imageStudioHistoryRepository) ListOldestByUser(ctx context.Context, userID int64, limit int) ([]service.ImageStudioHistoryRecord, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT `+imageStudioHistoryColumns+`
		FROM image_studio_history
		WHERE user_id = $1
		ORDER BY created_at ASC, id ASC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]service.ImageStudioHistoryRecord, 0, limit)
	for rows.Next() {
		rec, scanErr := scanImageStudioHistory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, *rec)
	}
	return records, rows.Err()
}

func (r *imageStudioHistoryRepository) ListExpired(ctx context.Context, before time.Time, limit int) ([]service.ImageStudioHistoryRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT `+imageStudioHistoryColumns+`
		FROM image_studio_history
		WHERE expires_at <= $1
		ORDER BY expires_at ASC
		LIMIT $2`, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]service.ImageStudioHistoryRecord, 0, limit)
	for rows.Next() {
		rec, scanErr := scanImageStudioHistory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, *rec)
	}
	return records, rows.Err()
}

func (r *imageStudioHistoryRepository) DeleteByIDs(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.sql.ExecContext(ctx,
		`DELETE FROM image_studio_history WHERE id = ANY($1)`, pq.Array(ids))
	return err
}

type imageStudioHistoryScanner interface {
	Scan(dest ...any) error
}

func scanImageStudioHistory(scanner imageStudioHistoryScanner) (*service.ImageStudioHistoryRecord, error) {
	var (
		rec       service.ImageStudioHistoryRecord
		apiKeyID  sql.NullInt64
		groupID   sql.NullInt64
		rawImages []byte
	)
	if err := scanner.Scan(
		&rec.ID, &rec.UserID, &apiKeyID, &groupID, &rec.Model, &rec.Prompt,
		&rec.RevisedPrompt, &rec.Size, &rawImages, &rec.ImageCount, &rec.CreatedAt, &rec.ExpiresAt,
	); err != nil {
		return nil, err
	}
	if apiKeyID.Valid {
		value := apiKeyID.Int64
		rec.APIKeyID = &value
	}
	if groupID.Valid {
		value := groupID.Int64
		rec.GroupID = &value
	}
	if len(rawImages) > 0 {
		if err := json.Unmarshal(rawImages, &rec.Images); err != nil {
			return nil, err
		}
	}
	if rec.ImageCount == 0 {
		rec.ImageCount = len(rec.Images)
	}
	return &rec, nil
}
