//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestResolveEffectiveRechargeMultiplier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userMult   *float64
		groupMult  *float64
		globalMult float64
		want       float64
	}{
		{name: "全部未设置回落全局", userMult: nil, groupMult: nil, globalMult: 1.5, want: 1.5},
		{name: "用户级优先", userMult: float64Ptr(2), groupMult: float64Ptr(3), globalMult: 1, want: 2},
		{name: "用户级为空时取分组级", userMult: nil, groupMult: float64Ptr(3), globalMult: 1, want: 3},
		{name: "用户级非法值视为未设置", userMult: float64Ptr(0), groupMult: float64Ptr(3), globalMult: 1, want: 3},
		{name: "分组级非法值回落全局", userMult: nil, groupMult: float64Ptr(-1), globalMult: 1.2, want: 1.2},
		{name: "用户级负数视为未设置", userMult: float64Ptr(-2), groupMult: nil, globalMult: 1.1, want: 1.1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, resolveEffectiveRechargeMultiplier(tt.userMult, tt.groupMult, tt.globalMult))
		})
	}
}

func newRechargeMultiplierTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:recharge_multiplier?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestMaxGroupRechargeMultiplier(t *testing.T) {
	client := newRechargeMultiplierTestClient(t)
	ctx := context.Background()
	svc := &PaymentService{entClient: client}

	// 空分组列表 / 无 ent client 时直接返回 nil
	maxMult, err := svc.maxGroupRechargeMultiplier(ctx, nil)
	require.NoError(t, err)
	require.Nil(t, maxMult)

	g1, err := client.Group.Create().SetName("rm-g1").SetRechargeMultiplier(1.5).Save(ctx)
	require.NoError(t, err)
	g2, err := client.Group.Create().SetName("rm-g2").SetRechargeMultiplier(2.5).Save(ctx)
	require.NoError(t, err)
	g3, err := client.Group.Create().SetName("rm-g3").Save(ctx) // 未设置倍率
	require.NoError(t, err)

	// 多组取最高倍率，未设置的组不参与
	maxMult, err = svc.maxGroupRechargeMultiplier(ctx, []int64{g1.ID, g2.ID, g3.ID})
	require.NoError(t, err)
	require.NotNil(t, maxMult)
	require.Equal(t, 2.5, *maxMult)

	// 全部未设置时返回 nil
	maxMult, err = svc.maxGroupRechargeMultiplier(ctx, []int64{g3.ID})
	require.NoError(t, err)
	require.Nil(t, maxMult)
}

func TestEffectiveRechargeMultiplier(t *testing.T) {
	client := newRechargeMultiplierTestClient(t)
	ctx := context.Background()
	svc := &PaymentService{entClient: client}

	grp, err := client.Group.Create().SetName("rm-eff-g1").SetRechargeMultiplier(2.0).Save(ctx)
	require.NoError(t, err)

	// nil 用户回落全局
	require.Equal(t, 1.0, svc.effectiveRechargeMultiplier(ctx, nil, 1.0))

	// 用户级优先于分组级
	u := &User{ID: 1, RechargeMultiplier: float64Ptr(3.0), AllowedGroups: []int64{grp.ID}}
	require.Equal(t, 3.0, svc.effectiveRechargeMultiplier(ctx, u, 1.0))

	// 用户级未设置时取分组级
	u = &User{ID: 1, AllowedGroups: []int64{grp.ID}}
	require.Equal(t, 2.0, svc.effectiveRechargeMultiplier(ctx, u, 1.0))

	// 分组级也未设置时回落全局
	u = &User{ID: 1}
	require.Equal(t, 1.3, svc.effectiveRechargeMultiplier(ctx, u, 1.3))
}

// 通过 user_allowed_groups 关联表验证完整的 GetEffectiveRechargeMultiplier 链路。
func TestGetEffectiveRechargeMultiplierViaAllowedGroups(t *testing.T) {
	client := newRechargeMultiplierTestClient(t)
	ctx := context.Background()

	grp, err := client.Group.Create().SetName("rm-full-g1").SetRechargeMultiplier(1.8).Save(ctx)
	require.NoError(t, err)

	userRepo := &mockUserRepo{getByIDUser: &User{ID: 7, AllowedGroups: []int64{grp.ID}}}
	svc := &PaymentService{entClient: client, userRepo: userRepo}
	require.Equal(t, 1.8, svc.GetEffectiveRechargeMultiplier(ctx, 7, 1.0))

	// 用户级覆盖后优先于分组级
	userRepo.getByIDUser.RechargeMultiplier = float64Ptr(2.2)
	require.Equal(t, 2.2, svc.GetEffectiveRechargeMultiplier(ctx, 7, 1.0))

	// 用户查询失败时回落全局
	userRepo.getByIDErr = ErrUserNotFound
	require.Equal(t, 1.0, svc.GetEffectiveRechargeMultiplier(ctx, 7, 1.0))
}
