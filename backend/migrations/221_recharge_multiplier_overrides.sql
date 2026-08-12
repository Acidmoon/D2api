-- 按用户/用户组的余额充值倍率覆盖。
-- 优先级：users.recharge_multiplier > groups.recharge_multiplier（多组取最高） > 全局 BALANCE_RECHARGE_MULTIPLIER。
-- NULL = 不覆盖，回落到下一级；合法值 > 0。
ALTER TABLE users ADD COLUMN IF NOT EXISTS recharge_multiplier DECIMAL(20,8);
ALTER TABLE groups ADD COLUMN IF NOT EXISTS recharge_multiplier DECIMAL(10,4);
