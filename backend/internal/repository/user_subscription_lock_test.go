package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

// TestUserSubscriptionGetByIDForUpdateReadsWalletTable 校验 GetByIDForUpdate
// 在我方订阅钱包架构下的数据源语义：订阅数据以 subscription_balances 表为准，
// 方法经由 listSubscriptionBalances 的 31 列原生 SQL 读取。
//
// 注意与上游的差异：上游实现对 ent user_subscriptions 表使用 SELECT ... FOR UPDATE
// 行锁；我方钱包路径整体走非事务原生 SQL（*sql.DB），不持有行锁，这是订阅钱包
// 架构的既定设计（续期等更新流程的并发一致性由钱包路径自身的条件更新保证）。
// 因此本测试断言查询命中 subscription_balances 且不含 FOR UPDATE，而不是上游的
// 行锁语义。
func TestUserSubscriptionGetByIDForUpdateReadsWalletTable(t *testing.T) {
	var capturedSQL string
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(captureEntQueryMatcher{actual: &capturedSQL}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))
	t.Cleanup(func() { _ = client.Close() })
	repo := NewUserSubscriptionRepository(client)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

	// 列顺序必须与 user_subscription_repo.go 中 listSubscriptionBalances 的
	// SELECT 列表一致（scanSubscriptionBalance 按位置 Scan）。
	columns := []string{
		"id", "user_id", "group_id", "source_group_id", "plan_name",
		"starts_at", "expires_at", "status",
		"daily_window_start", "weekly_window_start", "monthly_window_start",
		"daily_limit_usd", "weekly_limit_usd", "monthly_limit_usd",
		"daily_usage_usd", "weekly_usage_usd", "monthly_usage_usd",
		"assigned_by", "assigned_at", "notes", "created_at", "updated_at", "deleted_at",
		"user_email", "user_username",
		"group_name", "group_platform", "group_rate_multiplier", "group_subscription_type",
		"assigner_email", "assigner_username",
	}
	mock.ExpectQuery("wallet subscription").WillReturnRows(
		sqlmock.NewRows(columns).AddRow(
			int64(7), int64(11), int64(13), int64(13), "pro-plan",
			now, now.AddDate(0, 0, 30), "active",
			now, now, now,
			100.0, 200.0, 300.0,
			1.5, 2.5, 3.5,
			int64(42), now, "renewal note", now, now, nil,
			"user@example.com", "user11",
			"Pro Group", "openai", 1.5, "standard",
			"admin@example.com", "admin",
		),
	)

	sub, err := repo.GetByIDForUpdate(context.Background(), 7)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Equal(t, int64(7), sub.ID)
	require.Equal(t, int64(11), sub.UserID)
	require.Equal(t, int64(13), sub.GroupID)
	require.NotNil(t, sub.SourceGroupID)
	require.Equal(t, int64(13), *sub.SourceGroupID)
	require.Equal(t, "pro-plan", sub.PlanName)
	require.Equal(t, "active", sub.Status)
	require.NotNil(t, sub.DailyLimitUSD)
	require.Equal(t, 100.0, *sub.DailyLimitUSD)
	require.Equal(t, 1.5, sub.DailyUsageUSD)
	require.Equal(t, 2.5, sub.WeeklyUsageUSD)
	require.Equal(t, 3.5, sub.MonthlyUsageUSD)
	require.Equal(t, "renewal note", sub.Notes)
	require.NotNil(t, sub.AssignedBy)
	require.Equal(t, int64(42), *sub.AssignedBy)
	require.NotNil(t, sub.User)
	require.Equal(t, "user@example.com", sub.User.Email)
	require.NotNil(t, sub.Group)
	require.True(t, sub.Group.Hydrated)
	// Group.Name 优先取 plan_name（firstNonEmpty(planName, groupName)）。
	require.Equal(t, "pro-plan", sub.Group.Name)
	require.Equal(t, "standard", sub.Group.SubscriptionType)

	// 数据源断言：查询必须命中钱包表 subscription_balances，且不包含上游的
	// FOR UPDATE 行锁（钱包路径不持有行锁，见函数注释）。
	normalized := strings.ToUpper(normalizeSQLWhitespace(capturedSQL))
	require.Contains(t, normalized, "SUBSCRIPTION_BALANCES")
	require.NotContains(t, normalized, "FOR UPDATE")
}
