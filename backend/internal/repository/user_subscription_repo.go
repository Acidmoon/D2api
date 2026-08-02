package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type subscriptionBalanceExecutor interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type userSubscriptionRepository struct {
	client *dbent.Client
	// db 是对 subscription_balances 执行原生 SQL 的 executor。
	// 生产环境是 *sql.DB；测试可注入 *sql.Tx 以便与 ent 事务共享同一条事务。
	db subscriptionBalanceExecutor
}

func NewUserSubscriptionRepository(client *dbent.Client, dbOpt ...*sql.DB) service.UserSubscriptionRepository {
	var db *sql.DB
	if len(dbOpt) > 0 {
		db = dbOpt[0]
	}
	if db == nil && client != nil {
		if drv, ok := client.Driver().(*entsql.Driver); ok {
			db = drv.DB()
		}
	}
	// 注意：不能把类型化 nil *sql.DB 直接装入接口，否则 nil 判断会失效。
	var exec subscriptionBalanceExecutor
	if db != nil {
		exec = db
	}
	return &userSubscriptionRepository{client: client, db: exec}
}

func (r *userSubscriptionRepository) subscriptionBalanceExec() subscriptionBalanceExecutor {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db
}

func (r *userSubscriptionRepository) Create(ctx context.Context, sub *service.UserSubscription) error {
	if sub == nil {
		return service.ErrSubscriptionNilInput
	}

	if sub.StartsAt.IsZero() {
		sub.StartsAt = time.Now()
	}
	if sub.Status == "" {
		sub.Status = service.SubscriptionStatusActive
	}
	if sub.AssignedAt.IsZero() {
		sub.AssignedAt = time.Now()
	}
	if sub.PlanName == "" && sub.Group != nil {
		sub.PlanName = sub.Group.Name
	}
	return r.insertSubscriptionBalance(ctx, sub)
}

func (r *userSubscriptionRepository) GetByID(ctx context.Context, id int64) (*service.UserSubscription, error) {
	return r.getSubscriptionBalanceByID(ctx, id)
}

// GetByIDIncludeDeleted returns wallet history for administrative recovery.
func (r *userSubscriptionRepository) GetByIDIncludeDeleted(ctx context.Context, id int64) (*service.UserSubscription, error) {
	subs, err := r.listSubscriptionBalances(ctx, "WHERE sb.id = $1", []any{id}, "", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, service.ErrSubscriptionNotFound
	}
	return &subs[0], nil
}

func (r *userSubscriptionRepository) GetByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	return r.getSubscriptionBalanceByUserSourceGroup(ctx, userID, groupID, false)
}

func (r *userSubscriptionRepository) GetActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	return r.getSubscriptionBalanceByUserSourceGroup(ctx, userID, groupID, true)
}

func (r *userSubscriptionRepository) Update(ctx context.Context, sub *service.UserSubscription) error {
	if sub == nil {
		return service.ErrSubscriptionNilInput
	}

	return r.updateSubscriptionBalance(ctx, sub)
}

func (r *userSubscriptionRepository) Delete(ctx context.Context, id int64) error {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return errors.New("subscription balance repository db is nil")
	}
	_, err := exec.ExecContext(ctx, `
		WITH deleted AS (
			UPDATE subscription_balances
			SET deleted_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING legacy_user_subscription_id
		)
		UPDATE user_subscriptions us
		SET deleted_at = NOW(), updated_at = NOW()
		FROM deleted
		WHERE us.id = deleted.legacy_user_subscription_id
	`, id)
	return err
}

// Restore reactivates a soft-deleted wallet and its legacy compatibility row.
func (r *userSubscriptionRepository) Restore(ctx context.Context, subscriptionID int64, restoredStatus string) (*service.UserSubscription, error) {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return nil, errors.New("subscription balance repository db is nil")
	}
	result, err := exec.ExecContext(ctx, `
		WITH restored AS (
			UPDATE subscription_balances
			SET status = $1, deleted_at = NULL, updated_at = NOW()
			WHERE id = $2 AND deleted_at IS NOT NULL
			RETURNING legacy_user_subscription_id
		)
		UPDATE user_subscriptions us
		SET status = $1, deleted_at = NULL, updated_at = NOW()
		FROM restored
		WHERE us.id = restored.legacy_user_subscription_id
	`, restoredStatus, subscriptionID)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	// A group-less wallet has no legacy row, so confirm restoration from the
	// source-of-truth table when the compatibility UPDATE reports zero rows.
	if affected == 0 {
		var restored bool
		if err := exec.QueryRowContext(ctx, `SELECT deleted_at IS NULL FROM subscription_balances WHERE id = $1`, subscriptionID).Scan(&restored); err != nil || !restored {
			return nil, service.ErrSubscriptionNotFound
		}
	}
	return r.GetByID(ctx, subscriptionID)
}

func (r *userSubscriptionRepository) ListByUserID(ctx context.Context, userID int64) ([]service.UserSubscription, error) {
	return r.listSubscriptionBalances(ctx, "WHERE sb.user_id = $1 AND sb.deleted_at IS NULL", []any{userID}, "ORDER BY sb.created_at DESC", 0, 0)
}

func (r *userSubscriptionRepository) ListActiveByUserID(ctx context.Context, userID int64) ([]service.UserSubscription, error) {
	return r.listSubscriptionBalances(ctx, "WHERE sb.user_id = $1 AND sb.status = $2 AND sb.expires_at > NOW() AND sb.deleted_at IS NULL", []any{userID, service.SubscriptionStatusActive}, "ORDER BY sb.created_at DESC", 0, 0)
}

func (r *userSubscriptionRepository) ListByGroupID(ctx context.Context, groupID int64, params pagination.PaginationParams) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	where := "WHERE sb.source_group_id = $1 AND sb.deleted_at IS NULL"
	total, err := r.countSubscriptionBalances(ctx, where, []any{groupID})
	if err != nil {
		return nil, nil, err
	}
	subs, err := r.listSubscriptionBalances(ctx, where, []any{groupID}, "ORDER BY sb.created_at DESC", params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	return subs, paginationResultFromTotal(total, params), nil
}

func (r *userSubscriptionRepository) List(ctx context.Context, params pagination.PaginationParams, userID, groupID *int64, status, platform, sortBy, sortOrder string) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	includeSoftDeleted := status == "" || status == service.SubscriptionStatusRevoked
	clauses := make([]string, 0, 5)
	if !includeSoftDeleted {
		clauses = append(clauses, "sb.deleted_at IS NULL")
	}
	args := make([]any, 0, 4)
	if userID != nil {
		args = append(args, *userID)
		clauses = append(clauses, fmt.Sprintf("sb.user_id = $%d", len(args)))
	}
	if groupID != nil {
		args = append(args, *groupID)
		clauses = append(clauses, fmt.Sprintf("sb.source_group_id = $%d", len(args)))
	}
	if platform != "" {
		args = append(args, platform)
		clauses = append(clauses, fmt.Sprintf("g.platform = $%d", len(args)))
	}

	switch status {
	case service.SubscriptionStatusActive:
		args = append(args, service.SubscriptionStatusActive)
		clauses = append(clauses, fmt.Sprintf("sb.status = $%d AND sb.expires_at > NOW()", len(args)))
	case service.SubscriptionStatusExpired:
		clauses = append(clauses, "(sb.status = 'expired' OR (sb.status = 'active' AND sb.expires_at <= NOW()))")
	case service.SubscriptionStatusRevoked:
		clauses = append(clauses, "sb.deleted_at IS NOT NULL")
	case "":
	default:
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("sb.status = $%d", len(args)))
	}

	where := "WHERE TRUE"
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	total, err := r.countSubscriptionBalances(ctx, where, args)
	if err != nil {
		return nil, nil, err
	}

	field := "sb.created_at"
	switch sortBy {
	case "expires_at":
		field = "sb.expires_at"
	case "status":
		field = "sb.status"
	}
	order := "DESC"
	if sortOrder == "asc" && sortBy != "" {
		order = "ASC"
	}
	subs, err := r.listSubscriptionBalances(ctx, where, args, fmt.Sprintf("ORDER BY %s %s", field, order), params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	for i := range subs {
		if subs[i].DeletedAt != nil {
			subs[i].Status = service.SubscriptionStatusRevoked
		}
	}

	return subs, paginationResultFromTotal(total, params), nil
}

func (r *userSubscriptionRepository) ExistsByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (bool, error) {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return false, errors.New("subscription balance repository db is nil")
	}
	var exists bool
	err := exec.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM subscription_balances WHERE user_id = $1 AND source_group_id = $2 AND deleted_at IS NULL)`, userID, groupID).Scan(&exists)
	return exists, err
}

func (r *userSubscriptionRepository) ExistsActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (bool, error) {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return false, errors.New("subscription balance repository db is nil")
	}
	var exists bool
	err := exec.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM subscription_balances
		WHERE user_id = $1 AND source_group_id = $2 AND deleted_at IS NULL
			AND status = $3 AND expires_at > NOW()
	)`, userID, groupID, service.SubscriptionStatusActive).Scan(&exists)
	return exists, err
}

func (r *userSubscriptionRepository) ExtendExpiry(ctx context.Context, subscriptionID int64, newExpiresAt time.Time) error {
	if err := r.execSubscriptionBalanceUpdate(ctx, `UPDATE subscription_balances SET expires_at = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, newExpiresAt, subscriptionID); err != nil {
		return err
	}
	return r.syncLegacySubscriptionSnapshot(ctx, subscriptionID)
}

func (r *userSubscriptionRepository) UpdateStatus(ctx context.Context, subscriptionID int64, status string) error {
	if err := r.execSubscriptionBalanceUpdate(ctx, `UPDATE subscription_balances SET status = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, status, subscriptionID); err != nil {
		return err
	}
	return r.syncLegacySubscriptionSnapshot(ctx, subscriptionID)
}

func (r *userSubscriptionRepository) UpdateNotes(ctx context.Context, subscriptionID int64, notes string) error {
	if err := r.execSubscriptionBalanceUpdate(ctx, `UPDATE subscription_balances SET notes = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, notes, subscriptionID); err != nil {
		return err
	}
	return r.syncLegacySubscriptionSnapshot(ctx, subscriptionID)
}

func (r *userSubscriptionRepository) ActivateWindows(ctx context.Context, id int64, start time.Time) error {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return errors.New("subscription balance repository db is nil")
	}
	// 仅在三个窗口均未初始化时设置（首次激活语义，来自上游改进），
	// 避免后续请求覆盖已被其他请求推进的窗口起点。
	result, err := exec.ExecContext(ctx, `
		UPDATE subscription_balances
		SET daily_window_start = $1, weekly_window_start = $1, monthly_window_start = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
			AND daily_window_start IS NULL
			AND weekly_window_start IS NULL
			AND monthly_window_start IS NULL
	`, start, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		// 窗口已初始化是预期的 no-op；仅当订阅不存在时返回 not found。
		var exists bool
		if err := exec.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM subscription_balances WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return service.ErrSubscriptionNotFound
		}
		return nil
	}
	return r.syncLegacySubscriptionSnapshot(ctx, id)
}

func (r *userSubscriptionRepository) ResetUsageWindows(ctx context.Context, id int64, resetDaily, resetWeekly, resetMonthly bool, newWindowStart time.Time) error {
	if err := r.execSubscriptionBalanceUpdate(ctx, `
		UPDATE subscription_balances
		SET daily_usage_usd = CASE WHEN $1 THEN 0 ELSE daily_usage_usd END,
			daily_window_start = CASE WHEN $1 THEN $4 ELSE daily_window_start END,
			weekly_usage_usd = CASE WHEN $2 THEN 0 ELSE weekly_usage_usd END,
			weekly_window_start = CASE WHEN $2 THEN $4 ELSE weekly_window_start END,
			monthly_usage_usd = CASE WHEN $3 THEN 0 ELSE monthly_usage_usd END,
			monthly_window_start = CASE WHEN $3 THEN $4 ELSE monthly_window_start END,
			updated_at = NOW()
		WHERE id = $5 AND deleted_at IS NULL
	`, resetDaily, resetWeekly, resetMonthly, newWindowStart, id); err != nil {
		return err
	}
	return r.syncLegacySubscriptionSnapshot(ctx, id)
}

func (r *userSubscriptionRepository) ResetDailyUsage(ctx context.Context, id int64, expectedWindowStart *time.Time, newWindowStart time.Time) error {
	return r.resetUsageWindow(ctx, id, "daily_usage_usd", "daily_window_start", expectedWindowStart, newWindowStart)
}

func (r *userSubscriptionRepository) ResetWeeklyUsage(ctx context.Context, id int64, expectedWindowStart *time.Time, newWindowStart time.Time) error {
	return r.resetUsageWindow(ctx, id, "weekly_usage_usd", "weekly_window_start", expectedWindowStart, newWindowStart)
}

func (r *userSubscriptionRepository) ResetMonthlyUsage(ctx context.Context, id int64, expectedWindowStart *time.Time, newWindowStart time.Time) error {
	return r.resetUsageWindow(ctx, id, "monthly_usage_usd", "monthly_window_start", expectedWindowStart, newWindowStart)
}

// resetUsageWindow uses compare-and-swap semantics so an older request cannot
// reset usage after another request has already advanced the window.
func (r *userSubscriptionRepository) resetUsageWindow(ctx context.Context, id int64, usageColumn, windowColumn string, expectedWindowStart *time.Time, newWindowStart time.Time) error {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return errors.New("subscription balance repository db is nil")
	}
	query := fmt.Sprintf(`UPDATE subscription_balances SET %s = 0, %s = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL AND %s IS NOT DISTINCT FROM $3`, usageColumn, windowColumn, windowColumn)
	result, err := exec.ExecContext(ctx, query, newWindowStart, id, expectedWindowStart)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var exists bool
		if err := exec.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM subscription_balances WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return service.ErrSubscriptionNotFound
		}
		return nil
	}
	return r.syncLegacySubscriptionSnapshot(ctx, id)
}

// IncrementUsage 原子性地累加订阅用量。
// 限额检查已在请求前由 BillingCacheService.CheckBillingEligibility 完成，
// 此处仅负责记录实际消费，确保消费数据的完整性。
func (r *userSubscriptionRepository) IncrementUsage(ctx context.Context, id int64, costUSD float64) error {
	const updateSQL = `
		UPDATE subscription_balances sb
		SET
			daily_usage_usd = sb.daily_usage_usd + $1,
			weekly_usage_usd = sb.weekly_usage_usd + $1,
			monthly_usage_usd = sb.monthly_usage_usd + $1,
			updated_at = NOW()
		WHERE sb.id = $2
			AND sb.deleted_at IS NULL
	`

	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return errors.New("subscription balance repository db is nil")
	}
	result, err := exec.ExecContext(ctx, updateSQL, costUSD, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected > 0 {
		if err := r.syncLegacySubscriptionSnapshot(ctx, id); err != nil {
			return err
		}
		return nil
	}

	// affected == 0：订阅不存在或已删除
	return service.ErrSubscriptionNotFound
}

func (r *userSubscriptionRepository) BatchUpdateExpiredStatus(ctx context.Context) (int64, error) {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return 0, errors.New("subscription balance repository db is nil")
	}
	res, err := exec.ExecContext(ctx, `
		WITH updated AS (
			UPDATE subscription_balances
			SET status = $1, updated_at = NOW()
			WHERE status = $2 AND expires_at <= NOW() AND deleted_at IS NULL
			RETURNING legacy_user_subscription_id, status, updated_at
		)
		UPDATE user_subscriptions us
		SET status = updated.status, updated_at = updated.updated_at
		FROM updated
		WHERE us.id = updated.legacy_user_subscription_id
	`, service.SubscriptionStatusExpired, service.SubscriptionStatusActive)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return n, err
}

// Extra repository helpers (currently used only by integration tests).

func (r *userSubscriptionRepository) ListExpired(ctx context.Context) ([]service.UserSubscription, error) {
	return r.listSubscriptionBalances(ctx, "WHERE sb.status = $1 AND sb.expires_at <= NOW() AND sb.deleted_at IS NULL", []any{service.SubscriptionStatusActive}, "ORDER BY sb.expires_at ASC", 0, 0)
}

func (r *userSubscriptionRepository) CountByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return r.countSubscriptionBalances(ctx, "WHERE sb.source_group_id = $1 AND sb.deleted_at IS NULL", []any{groupID})
}

func (r *userSubscriptionRepository) CountActiveByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return r.countSubscriptionBalances(ctx, "WHERE sb.source_group_id = $1 AND sb.status = $2 AND sb.expires_at > NOW() AND sb.deleted_at IS NULL", []any{groupID, service.SubscriptionStatusActive})
}

func (r *userSubscriptionRepository) DeleteByGroupID(ctx context.Context, groupID int64) (int64, error) {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return 0, errors.New("subscription balance repository db is nil")
	}
	res, err := exec.ExecContext(ctx, `UPDATE subscription_balances SET deleted_at = NOW(), updated_at = NOW() WHERE source_group_id = $1 AND deleted_at IS NULL`, groupID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func userSubscriptionEntityToService(m *dbent.UserSubscription) *service.UserSubscription {
	if m == nil {
		return nil
	}
	out := &service.UserSubscription{
		ID:                 m.ID,
		UserID:             m.UserID,
		GroupID:            m.GroupID,
		StartsAt:           m.StartsAt,
		ExpiresAt:          m.ExpiresAt,
		Status:             m.Status,
		DailyWindowStart:   m.DailyWindowStart,
		WeeklyWindowStart:  m.WeeklyWindowStart,
		MonthlyWindowStart: m.MonthlyWindowStart,
		DailyUsageUSD:      m.DailyUsageUsd,
		WeeklyUsageUSD:     m.WeeklyUsageUsd,
		MonthlyUsageUSD:    m.MonthlyUsageUsd,
		AssignedBy:         m.AssignedBy,
		AssignedAt:         m.AssignedAt,
		Notes:              derefString(m.Notes),
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
	if m.Edges.User != nil {
		out.User = userEntityToService(m.Edges.User)
	}
	if m.Edges.Group != nil {
		out.Group = groupEntityToService(m.Edges.Group)
	}
	if m.Edges.AssignedByUser != nil {
		out.AssignedByUser = userEntityToService(m.Edges.AssignedByUser)
	}
	return out
}

func (r *userSubscriptionRepository) insertSubscriptionBalance(ctx context.Context, sub *service.UserSubscription) error {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return errors.New("subscription balance repository db is nil")
	}
	var id int64
	if sub.SourceGroupID == nil && sub.GroupID <= 0 {
		err := exec.QueryRowContext(ctx, `
			INSERT INTO subscription_balances (
				user_id, source_group_id, plan_name, source, starts_at, expires_at, status,
				daily_limit_usd, weekly_limit_usd, monthly_limit_usd,
				daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
				daily_window_start, weekly_window_start, monthly_window_start,
				assigned_by, assigned_at, notes, created_at, updated_at
			)
			VALUES (
				$1, NULL, $2, 'manual', $3, $4, $5,
				$6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW(), NOW()
			)
			RETURNING id, created_at, updated_at
		`,
			sub.UserID, sub.PlanName,
			sub.StartsAt, sub.ExpiresAt, sub.Status,
			sub.DailyLimitUSD, sub.WeeklyLimitUSD, sub.MonthlyLimitUSD,
			sub.DailyUsageUSD, sub.WeeklyUsageUSD, sub.MonthlyUsageUSD,
			sub.DailyWindowStart, sub.WeeklyWindowStart, sub.MonthlyWindowStart,
			sub.AssignedBy, sub.AssignedAt, nullableString(sub.Notes),
		).Scan(&id, &sub.CreatedAt, &sub.UpdatedAt)
		if err != nil {
			return translatePersistenceError(err, nil, service.ErrSubscriptionAlreadyExists)
		}
		sub.ID = id
		return nil
	}

	err := exec.QueryRowContext(ctx, `
		WITH legacy AS (
			INSERT INTO user_subscriptions (
				user_id, group_id, starts_at, expires_at, status,
				daily_window_start, weekly_window_start, monthly_window_start,
				daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
				assigned_by, assigned_at, notes, created_at, updated_at
			)
			VALUES ($1, $2, $4, $5, $6, $13, $14, $15, $10, $11, $12, $16, $17, $18, NOW(), NOW())
			RETURNING id, created_at, updated_at
		),
		balance AS (
			INSERT INTO subscription_balances (
				id, user_id, legacy_user_subscription_id, source_group_id, plan_name, source, starts_at, expires_at, status,
				daily_limit_usd, weekly_limit_usd, monthly_limit_usd,
				daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
				daily_window_start, weekly_window_start, monthly_window_start,
				assigned_by, assigned_at, notes, created_at, updated_at
			)
			SELECT
				legacy.id, $1, legacy.id, $2, $3, 'manual', $4, $5, $6,
				$7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
				legacy.created_at, legacy.updated_at
			FROM legacy
			RETURNING id, created_at, updated_at
		)
		SELECT id, created_at, updated_at FROM balance
	`,
		sub.UserID, nullableInt64(sub.SourceGroupID, sub.GroupID), sub.PlanName,
		sub.StartsAt, sub.ExpiresAt, sub.Status,
		sub.DailyLimitUSD, sub.WeeklyLimitUSD, sub.MonthlyLimitUSD,
		sub.DailyUsageUSD, sub.WeeklyUsageUSD, sub.MonthlyUsageUSD,
		sub.DailyWindowStart, sub.WeeklyWindowStart, sub.MonthlyWindowStart,
		sub.AssignedBy, sub.AssignedAt, nullableString(sub.Notes),
	).Scan(&id, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrSubscriptionAlreadyExists)
	}
	sub.ID = id
	return nil
}

func (r *userSubscriptionRepository) updateSubscriptionBalance(ctx context.Context, sub *service.UserSubscription) error {
	err := r.execSubscriptionBalanceUpdate(ctx, `
		UPDATE subscription_balances
		SET user_id = $1,
			source_group_id = $2,
			plan_name = $3,
			starts_at = $4,
			expires_at = $5,
			status = $6,
			daily_limit_usd = $7,
			weekly_limit_usd = $8,
			monthly_limit_usd = $9,
			daily_usage_usd = $10,
			weekly_usage_usd = $11,
			monthly_usage_usd = $12,
			daily_window_start = $13,
			weekly_window_start = $14,
			monthly_window_start = $15,
			assigned_by = $16,
			assigned_at = $17,
			notes = $18,
			updated_at = NOW()
		WHERE id = $19 AND deleted_at IS NULL
	`,
		sub.UserID, nullableInt64(sub.SourceGroupID, sub.GroupID), sub.PlanName,
		sub.StartsAt, sub.ExpiresAt, sub.Status,
		sub.DailyLimitUSD, sub.WeeklyLimitUSD, sub.MonthlyLimitUSD,
		sub.DailyUsageUSD, sub.WeeklyUsageUSD, sub.MonthlyUsageUSD,
		sub.DailyWindowStart, sub.WeeklyWindowStart, sub.MonthlyWindowStart,
		sub.AssignedBy, sub.AssignedAt, nullableString(sub.Notes), sub.ID,
	)
	if err != nil {
		return err
	}
	return r.syncLegacySubscriptionSnapshot(ctx, sub.ID)
}

func (r *userSubscriptionRepository) getSubscriptionBalanceByID(ctx context.Context, id int64) (*service.UserSubscription, error) {
	subs, err := r.listSubscriptionBalances(ctx, "WHERE sb.id = $1 AND sb.deleted_at IS NULL", []any{id}, "", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, service.ErrSubscriptionNotFound
	}
	return &subs[0], nil
}

func (r *userSubscriptionRepository) getSubscriptionBalanceByUserSourceGroup(ctx context.Context, userID, groupID int64, activeOnly bool) (*service.UserSubscription, error) {
	where := "WHERE sb.user_id = $1 AND sb.source_group_id = $2 AND sb.deleted_at IS NULL"
	args := []any{userID, groupID}
	if activeOnly {
		where += " AND sb.status = $3 AND sb.expires_at > NOW()"
		args = append(args, service.SubscriptionStatusActive)
	}
	subs, err := r.listSubscriptionBalances(ctx, where, args, "ORDER BY sb.created_at DESC", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, service.ErrSubscriptionNotFound
	}
	return &subs[0], nil
}

func (r *userSubscriptionRepository) countSubscriptionBalances(ctx context.Context, where string, args []any) (int64, error) {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return 0, errors.New("subscription balance repository db is nil")
	}
	var total int64
	err := exec.QueryRowContext(ctx, "SELECT COUNT(*) FROM subscription_balances sb LEFT JOIN groups g ON g.id = sb.source_group_id "+where, args...).Scan(&total)
	return total, err
}

func (r *userSubscriptionRepository) listSubscriptionBalances(ctx context.Context, where string, args []any, order string, limit, offset int) ([]service.UserSubscription, error) {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return nil, errors.New("subscription balance repository db is nil")
	}
	query := `
		SELECT
			sb.id, sb.user_id, COALESCE(sb.source_group_id, 0), sb.source_group_id, sb.plan_name,
			sb.starts_at, sb.expires_at, sb.status,
			sb.daily_window_start, sb.weekly_window_start, sb.monthly_window_start,
			sb.daily_limit_usd, sb.weekly_limit_usd, sb.monthly_limit_usd,
			sb.daily_usage_usd, sb.weekly_usage_usd, sb.monthly_usage_usd,
			sb.assigned_by, sb.assigned_at, sb.notes, sb.created_at, sb.updated_at, sb.deleted_at,
			u.email, u.username,
			g.name, g.platform, g.rate_multiplier, g.subscription_type,
			au.email, au.username
		FROM subscription_balances sb
		LEFT JOIN users u ON u.id = sb.user_id
		LEFT JOIN groups g ON g.id = sb.source_group_id
		LEFT JOIN users au ON au.id = sb.assigned_by
		` + where + " " + order
	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []service.UserSubscription
	for rows.Next() {
		sub, err := scanSubscriptionBalance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, rows.Err()
}

func scanSubscriptionBalance(rows *sql.Rows) (*service.UserSubscription, error) {
	var (
		sub                   service.UserSubscription
		sourceGroupID         sql.NullInt64
		planName              string
		dailyStart            sql.NullTime
		weeklyStart           sql.NullTime
		monthlyStart          sql.NullTime
		dailyLimit            sql.NullFloat64
		weeklyLimit           sql.NullFloat64
		monthlyLimit          sql.NullFloat64
		assignedBy            sql.NullInt64
		notes                 sql.NullString
		userEmail             sql.NullString
		userName              sql.NullString
		groupName             sql.NullString
		groupPlatform         sql.NullString
		groupRate             sql.NullFloat64
		groupSubscriptionType sql.NullString
		deletedAt             sql.NullTime
		assignerEmail         sql.NullString
		assignerName          sql.NullString
	)
	if err := rows.Scan(
		&sub.ID, &sub.UserID, &sub.GroupID, &sourceGroupID, &planName,
		&sub.StartsAt, &sub.ExpiresAt, &sub.Status,
		&dailyStart, &weeklyStart, &monthlyStart,
		&dailyLimit, &weeklyLimit, &monthlyLimit,
		&sub.DailyUsageUSD, &sub.WeeklyUsageUSD, &sub.MonthlyUsageUSD,
		&assignedBy, &sub.AssignedAt, &notes, &sub.CreatedAt, &sub.UpdatedAt, &deletedAt,
		&userEmail, &userName,
		&groupName, &groupPlatform, &groupRate, &groupSubscriptionType,
		&assignerEmail, &assignerName,
	); err != nil {
		return nil, err
	}
	sub.PlanName = planName
	sub.SourceGroupID = nullableInt64Ptr(sourceGroupID)
	sub.DailyWindowStart = nullableTimePtr(dailyStart)
	sub.WeeklyWindowStart = nullableTimePtr(weeklyStart)
	sub.MonthlyWindowStart = nullableTimePtr(monthlyStart)
	sub.DailyLimitUSD = subscriptionNullableFloat64Ptr(dailyLimit)
	sub.WeeklyLimitUSD = subscriptionNullableFloat64Ptr(weeklyLimit)
	sub.MonthlyLimitUSD = subscriptionNullableFloat64Ptr(monthlyLimit)
	sub.AssignedBy = nullableInt64Ptr(assignedBy)
	if sub.AssignedBy != nil && (assignerEmail.Valid || assignerName.Valid) {
		sub.AssignedByUser = &service.User{ID: *sub.AssignedBy, Email: assignerEmail.String, Username: assignerName.String}
	}
	sub.Notes = derefStringFromNull(notes)
	sub.DeletedAt = nullableTimePtr(deletedAt)
	if userEmail.Valid || userName.Valid {
		sub.User = &service.User{ID: sub.UserID, Email: userEmail.String, Username: userName.String}
	}
	if sourceGroupID.Valid {
		sub.Group = &service.Group{
			ID:               sourceGroupID.Int64,
			Name:             firstNonEmpty(planName, groupName.String),
			Platform:         groupPlatform.String,
			RateMultiplier:   groupRate.Float64,
			SubscriptionType: groupSubscriptionType.String,
			DailyLimitUSD:    sub.DailyLimitUSD,
			WeeklyLimitUSD:   sub.WeeklyLimitUSD,
			MonthlyLimitUSD:  sub.MonthlyLimitUSD,
			Hydrated:         true,
		}
	}
	return &sub, nil
}

func (r *userSubscriptionRepository) execSubscriptionBalanceUpdate(ctx context.Context, query string, args ...any) error {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return errors.New("subscription balance repository db is nil")
	}
	res, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrSubscriptionNotFound
	}
	return nil
}

func (r *userSubscriptionRepository) syncLegacySubscriptionSnapshot(ctx context.Context, subscriptionID int64) error {
	exec := r.subscriptionBalanceExec()
	if exec == nil {
		return errors.New("subscription balance repository db is nil")
	}
	_, err := exec.ExecContext(ctx, `
		UPDATE user_subscriptions us
		SET user_id = sb.user_id,
			group_id = COALESCE(sb.source_group_id, us.group_id),
			starts_at = sb.starts_at,
			expires_at = sb.expires_at,
			status = sb.status,
			daily_window_start = sb.daily_window_start,
			weekly_window_start = sb.weekly_window_start,
			monthly_window_start = sb.monthly_window_start,
			daily_usage_usd = sb.daily_usage_usd,
			weekly_usage_usd = sb.weekly_usage_usd,
			monthly_usage_usd = sb.monthly_usage_usd,
			assigned_by = sb.assigned_by,
			assigned_at = sb.assigned_at,
			notes = sb.notes,
			updated_at = sb.updated_at
		FROM subscription_balances sb
		WHERE sb.id = $1
			AND us.id = sb.legacy_user_subscription_id
	`, subscriptionID)
	return err
}

func nullableInt64(ptr *int64, fallback int64) any {
	if ptr != nil {
		return *ptr
	}
	if fallback > 0 {
		return fallback
	}
	return nil
}

func nullableString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullableInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}

func nullableTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	x := v.Time
	return &x
}

func subscriptionNullableFloat64Ptr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	x := v.Float64
	return &x
}

func derefStringFromNull(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
