-- 生图创作中心历史：每次生成一行，图片字节落在 <image_studio.data_dir>/<user_id>/<dir>/<index>.<ext>，
-- 本表只存元数据与相对路径。expires_at 固定为 created_at + retention_days（默认 7 天），
-- 由 image_studio_history 清理 worker 删除过期记录与文件。
CREATE TABLE IF NOT EXISTS image_studio_history (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT       NOT NULL,
    api_key_id     BIGINT,
    group_id       BIGINT,
    model          VARCHAR(128) NOT NULL,
    prompt         TEXT         NOT NULL,
    revised_prompt TEXT,
    size           VARCHAR(32),
    -- [{"file":"<dir>/0.png","mime":"image/png","bytes":12345}]
    images         JSONB        NOT NULL DEFAULT '[]'::jsonb,
    image_count    INTEGER      NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    expires_at     TIMESTAMPTZ  NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_image_studio_history_user_created
    ON image_studio_history (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_image_studio_history_expires
    ON image_studio_history (expires_at);
