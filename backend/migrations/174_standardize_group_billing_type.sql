-- Subscription wallets are user-scoped. Route groups are no longer a billing
-- selector, so normalize legacy rows and prevent new subscription-type groups.
UPDATE groups
SET subscription_type = 'standard'
WHERE subscription_type IS DISTINCT FROM 'standard';

ALTER TABLE groups
    DROP CONSTRAINT IF EXISTS groups_subscription_type_check;

ALTER TABLE groups
    ADD CONSTRAINT groups_subscription_type_check
    CHECK (subscription_type = 'standard');
