-- 153: Convert subscriptions from group authorization to user-level prepaid wallet semantics.
--
-- The migration intentionally keeps user_subscriptions rows unchanged: user_id,
-- group_id, starts/expires/status, usage windows, usage amounts, assignment data,
-- and notes remain the source of truth for existing APC subscribers.
-- New application code interprets active user_subscriptions as user-level
-- subscription wallets and spends them before user balance.

CREATE TABLE IF NOT EXISTS subscription_wallet_migration_audit (
    id BIGSERIAL PRIMARY KEY,
    migration_key VARCHAR(100) NOT NULL,
    user_subscription_id BIGINT NOT NULL REFERENCES user_subscriptions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    group_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    daily_usage_usd DECIMAL(20,10) NOT NULL,
    weekly_usage_usd DECIMAL(20,10) NOT NULL,
    monthly_usage_usd DECIMAL(20,10) NOT NULL,
    daily_window_start TIMESTAMPTZ,
    weekly_window_start TIMESTAMPTZ,
    monthly_window_start TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS subscription_wallet_migration_audit_key_sub
    ON subscription_wallet_migration_audit(migration_key, user_subscription_id);

INSERT INTO subscription_wallet_migration_audit (
    migration_key,
    user_subscription_id,
    user_id,
    group_id,
    group_name,
    status,
    starts_at,
    expires_at,
    daily_usage_usd,
    weekly_usage_usd,
    monthly_usage_usd,
    daily_window_start,
    weekly_window_start,
    monthly_window_start
)
SELECT
    '153_apc_subscription_wallet',
    us.id,
    us.user_id,
    us.group_id,
    g.name,
    us.status,
    us.starts_at,
    us.expires_at,
    us.daily_usage_usd,
    us.weekly_usage_usd,
    us.monthly_usage_usd,
    us.daily_window_start,
    us.weekly_window_start,
    us.monthly_window_start
FROM user_subscriptions us
JOIN groups g ON g.id = us.group_id AND g.deleted_at IS NULL
WHERE us.deleted_at IS NULL
  AND g.subscription_type = 'subscription'
  AND (
      g.name ILIKE '%APC%'
      OR g.description ILIKE '%APC%'
      OR g.platform = 'anthropic'
  )
ON CONFLICT (migration_key, user_subscription_id) DO NOTHING;

COMMENT ON TABLE subscription_wallet_migration_audit IS 'Audit snapshot for converting existing APC/group subscriptions into user-level subscription wallet semantics without rewriting subscription state or usage.';
