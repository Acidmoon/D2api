-- 154: Move runtime subscriptions out of routing groups.
--
-- subscription_balances is the new source of truth for subscription billing.
-- Legacy user_subscriptions and APC subscription groups are read only as a
-- migration source; runtime billing no longer joins groups for subscription
-- limits or eligibility.

CREATE TABLE IF NOT EXISTS subscription_balances (
    id BIGINT PRIMARY KEY DEFAULT nextval('user_subscriptions_id_seq'::regclass),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    legacy_user_subscription_id BIGINT UNIQUE REFERENCES user_subscriptions(id) ON DELETE SET NULL,
    source_group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    plan_name VARCHAR(100) NOT NULL DEFAULT '',
    source VARCHAR(50) NOT NULL DEFAULT 'legacy_apc',
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL,
    daily_limit_usd DECIMAL(20,10),
    weekly_limit_usd DECIMAL(20,10),
    monthly_limit_usd DECIMAL(20,10),
    daily_usage_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    weekly_usage_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    monthly_usage_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    daily_window_start TIMESTAMPTZ,
    weekly_window_start TIMESTAMPTZ,
    monthly_window_start TIMESTAMPTZ,
    assigned_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    migrated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_subscription_balances_user_id
    ON subscription_balances(user_id);
CREATE INDEX IF NOT EXISTS idx_subscription_balances_user_status_expires
    ON subscription_balances(user_id, status, expires_at)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_subscription_balances_source_group_id
    ON subscription_balances(source_group_id);

ALTER TABLE subscription_balances
    ALTER COLUMN id SET DEFAULT nextval('user_subscriptions_id_seq'::regclass);

INSERT INTO subscription_balances (
    id,
    user_id,
    legacy_user_subscription_id,
    source_group_id,
    plan_name,
    source,
    starts_at,
    expires_at,
    status,
    daily_limit_usd,
    weekly_limit_usd,
    monthly_limit_usd,
    daily_usage_usd,
    weekly_usage_usd,
    monthly_usage_usd,
    daily_window_start,
    weekly_window_start,
    monthly_window_start,
    assigned_by,
    assigned_at,
    notes,
    created_at,
    updated_at,
    deleted_at
)
SELECT
    us.id,
    us.user_id,
    us.id,
    us.group_id,
    g.name,
    'legacy_apc',
    us.starts_at,
    us.expires_at,
    us.status,
    g.daily_limit_usd,
    g.weekly_limit_usd,
    g.monthly_limit_usd,
    us.daily_usage_usd,
    us.weekly_usage_usd,
    us.monthly_usage_usd,
    us.daily_window_start,
    us.weekly_window_start,
    us.monthly_window_start,
    us.assigned_by,
    us.assigned_at,
    us.notes,
    us.created_at,
    us.updated_at,
    us.deleted_at
FROM user_subscriptions us
JOIN groups g ON g.id = us.group_id AND g.deleted_at IS NULL
WHERE us.deleted_at IS NULL
  AND g.subscription_type = 'subscription'
  AND (
      g.name ILIKE '%APC%'
      OR g.description ILIKE '%APC%'
      OR g.platform = 'anthropic'
  )
ON CONFLICT (id) DO UPDATE
SET
    user_id = EXCLUDED.user_id,
    legacy_user_subscription_id = EXCLUDED.legacy_user_subscription_id,
    source_group_id = EXCLUDED.source_group_id,
    plan_name = EXCLUDED.plan_name,
    source = EXCLUDED.source,
    starts_at = EXCLUDED.starts_at,
    expires_at = EXCLUDED.expires_at,
    status = EXCLUDED.status,
    daily_limit_usd = EXCLUDED.daily_limit_usd,
    weekly_limit_usd = EXCLUDED.weekly_limit_usd,
    monthly_limit_usd = EXCLUDED.monthly_limit_usd,
    daily_usage_usd = EXCLUDED.daily_usage_usd,
    weekly_usage_usd = EXCLUDED.weekly_usage_usd,
    monthly_usage_usd = EXCLUDED.monthly_usage_usd,
    daily_window_start = EXCLUDED.daily_window_start,
    weekly_window_start = EXCLUDED.weekly_window_start,
    monthly_window_start = EXCLUDED.monthly_window_start,
    assigned_by = EXCLUDED.assigned_by,
    assigned_at = EXCLUDED.assigned_at,
    notes = EXCLUDED.notes,
    updated_at = EXCLUDED.updated_at,
    deleted_at = EXCLUDED.deleted_at;

SELECT setval(
    pg_get_serial_sequence('user_subscriptions', 'id'),
    GREATEST(
        COALESCE((SELECT MAX(id) FROM user_subscriptions), 0),
        COALESCE((SELECT MAX(id) FROM subscription_balances), 0),
        1
    ),
    true
);

ALTER TABLE usage_logs DROP CONSTRAINT IF EXISTS usage_logs_user_subscriptions_usage_logs;
ALTER TABLE usage_logs DROP CONSTRAINT IF EXISTS usage_logs_subscription_id_fkey;
UPDATE usage_logs ul
SET subscription_id = NULL
WHERE ul.subscription_id IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM subscription_balances sb
      WHERE sb.id = ul.subscription_id
  );
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'usage_logs_subscription_balances_usage_logs'
    ) THEN
        ALTER TABLE usage_logs
            ADD CONSTRAINT usage_logs_subscription_balances_usage_logs
            FOREIGN KEY (subscription_id) REFERENCES subscription_balances(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF to_regclass('public.billing_usage_entries') IS NOT NULL THEN
        ALTER TABLE billing_usage_entries DROP CONSTRAINT IF EXISTS billing_usage_entries_subscription_id_fkey;

        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'billing_usage_entries_subscription_balances_fkey'
        ) THEN
            ALTER TABLE billing_usage_entries
                ADD CONSTRAINT billing_usage_entries_subscription_balances_fkey
                FOREIGN KEY (subscription_id) REFERENCES subscription_balances(id) ON DELETE SET NULL;
        END IF;
    END IF;
END $$;

COMMENT ON TABLE subscription_balances IS 'User-level subscription balance wallets. Runtime billing uses this table instead of subscription groups.';
