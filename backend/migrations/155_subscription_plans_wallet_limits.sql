-- 155: Store subscription wallet limits directly on payment plans.
--
-- Plans may still reference a legacy group for display/backward compatibility,
-- but new subscription balances must be fulfilled from these plan-level quota
-- fields instead of using subscription groups as the subscription source.

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS daily_limit_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS weekly_limit_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS monthly_limit_usd DECIMAL(20,10);

UPDATE subscription_plans sp
SET
    daily_limit_usd = COALESCE(sp.daily_limit_usd, g.daily_limit_usd),
    weekly_limit_usd = COALESCE(sp.weekly_limit_usd, g.weekly_limit_usd),
    monthly_limit_usd = COALESCE(sp.monthly_limit_usd, g.monthly_limit_usd)
FROM groups g
WHERE sp.group_id = g.id;

COMMENT ON COLUMN subscription_plans.daily_limit_usd IS 'Daily subscription wallet quota in USD. NULL or <= 0 means unlimited.';
COMMENT ON COLUMN subscription_plans.weekly_limit_usd IS 'Weekly subscription wallet quota in USD. NULL or <= 0 means unlimited.';
COMMENT ON COLUMN subscription_plans.monthly_limit_usd IS 'Monthly subscription wallet quota in USD. NULL or <= 0 means unlimited.';
