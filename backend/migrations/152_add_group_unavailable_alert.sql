-- Add per-group unavailable alert switch.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS unavailable_alert_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN groups.unavailable_alert_enabled IS '是否在该分组所有账号不可用时发送邮件警告';
