package service

import (
	"context"
	"log/slog"

	entgroup "github.com/Wei-Shaw/sub2api/ent/group"
)

// resolveEffectiveRechargeMultiplier 计算有效余额充值倍率：
// 用户级字段 > 分组级字段 > 全局设置；nil 或 <=0 视为未设置，回落到下一级。
func resolveEffectiveRechargeMultiplier(userMult, groupMult *float64, globalMult float64) float64 {
	if userMult != nil && *userMult > 0 {
		return *userMult
	}
	if groupMult != nil && *groupMult > 0 {
		return *groupMult
	}
	return globalMult
}

// maxGroupRechargeMultiplier 查询用户 allowed_groups 中非空 recharge_multiplier 的最高值。
// 无分组或所有分组均未设置时返回 nil。直接查询、不加缓存：下单与结账预览都是低频路径。
func (s *PaymentService) maxGroupRechargeMultiplier(ctx context.Context, groupIDs []int64) (*float64, error) {
	if len(groupIDs) == 0 || s.entClient == nil {
		return nil, nil
	}
	rows, err := s.entClient.Group.Query().
		Where(
			entgroup.IDIn(groupIDs...),
			entgroup.RechargeMultiplierNotNil(),
		).
		Select(entgroup.FieldRechargeMultiplier).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var maxMult *float64
	for _, row := range rows {
		if row.RechargeMultiplier == nil || *row.RechargeMultiplier <= 0 {
			continue
		}
		if maxMult == nil || *row.RechargeMultiplier > *maxMult {
			maxMult = row.RechargeMultiplier
		}
	}
	return maxMult, nil
}

// effectiveRechargeMultiplier 返回该用户当前的有效充值倍率（用户级 > 分组级 > 全局）。
// 分组查询失败时回落到全局设置并记日志，不阻塞下单。
func (s *PaymentService) effectiveRechargeMultiplier(ctx context.Context, user *User, globalMult float64) float64 {
	if user == nil {
		return globalMult
	}
	if user.RechargeMultiplier != nil && *user.RechargeMultiplier > 0 {
		return *user.RechargeMultiplier
	}
	groupMult, err := s.maxGroupRechargeMultiplier(ctx, user.AllowedGroups)
	if err != nil {
		slog.Warn("[PaymentService] resolve group recharge multiplier failed, fallback to global",
			"user_id", user.ID, "error", err)
		return globalMult
	}
	return resolveEffectiveRechargeMultiplier(nil, groupMult, globalMult)
}

// GetEffectiveRechargeMultiplier 供结账预览等外部调用方按用户 ID 查询有效充值倍率。
// 用户不存在或查询失败时回落到全局设置。
func (s *PaymentService) GetEffectiveRechargeMultiplier(ctx context.Context, userID int64, globalMult float64) float64 {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		slog.Warn("[PaymentService] get user for recharge multiplier failed, fallback to global",
			"user_id", userID, "error", err)
		return globalMult
	}
	return s.effectiveRechargeMultiplier(ctx, user, globalMult)
}
