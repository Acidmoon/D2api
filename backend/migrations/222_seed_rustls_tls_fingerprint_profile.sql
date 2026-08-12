-- Seed a rustls (Codex CLI / reqwest+rustls) TLS fingerprint profile.
--
-- 背景：OpenAI 平台账号的 UA/originator 宣称 codex-tui（Rust codex-rs，reqwest+rustls），
-- 而此前唯一的内置模板是 Node.js/BoringSSL 指纹，存在跨层矛盾。本种子把 rustls
-- 模板入库，供账号在管理界面显式绑定；未绑定模板（profile_id=0）的 OpenAI 账号
-- 由代码内置的 DefaultRustlsProfile 兜底（见 service.ResolveTLSProfile）。
--
-- 指纹依据（与 backend/internal/pkg/tlsfingerprint/dialer.go 中 rustls* 常量一致）：
--   - rustls 0.23 源码 client/hs.rs 的扩展构造顺序
--   - ring provider 默认 cipher suites / kx groups / verify schemes 顺序
--   - rustls 不发送 GREASE；TLS1.2 重协商以 SCSV（255）附在 cipher 列表末尾
--
-- 说明：rustls 0.23.21+ 会在发送前随机重排扩展顺序；DB 模板存的是固定构造顺序
-- （不随机化），代码内置模板的 ShuffleExtensions 才会逐握手随机。JA4 对扩展顺序
-- 不敏感，两种形态都能对上 rustls 的 JA4。
--
-- 幂等：按 name 唯一约束去重，可重复执行。

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

INSERT INTO tls_fingerprint_profiles (
    name, description, enable_grease,
    cipher_suites, curves, point_formats, signature_algorithms,
    alpn_protocols, supported_versions, key_share_groups, psk_modes, extensions
)
SELECT
    'rustls (Codex CLI / reqwest)',
    'rustls 0.23 (ring provider) 默认 ClientHello，匹配 Codex CLI（codex-rs / reqwest）的 TLS 指纹。不发送 GREASE，cipher 列表末尾带 SCSV(255)，ALPN 为 h2+http/1.1。适用于 OpenAI 平台账号。',
    false,
    -- TLS1.3: AES_256_GCM/CHACHA20/AES_128_GCM 在前（rustls 默认序），末尾 SCSV
    '[4866, 4867, 4865, 49195, 49199, 49196, 49200, 52393, 52392, 255]'::jsonb,
    '[29, 23, 24]'::jsonb,           -- X25519, secp256r1, secp384r1
    '[0]'::jsonb,                    -- uncompressed
    -- webpki verifier 默认 supported_verify_schemes 顺序
    '[1027, 1283, 1539, 2055, 2054, 2053, 2052, 1537, 1281, 1025]'::jsonb,
    '["h2", "http/1.1"]'::jsonb,
    '[772, 771]'::jsonb,             -- TLS1.3, TLS1.2
    '[29]'::jsonb,                   -- key_share 仅 X25519
    '[1]'::jsonb,                    -- psk_dhe_ke
    -- rustls 扩展构造顺序：supported_versions, supported_groups, signature_algorithms,
    -- extended_master_secret, status_request, ec_point_formats, server_name,
    -- key_share, psk_key_exchange_modes, alpn, session_ticket
    '[43, 10, 13, 23, 5, 11, 0, 51, 45, 16, 35]'::jsonb
WHERE NOT EXISTS (
    SELECT 1 FROM tls_fingerprint_profiles WHERE name = 'rustls (Codex CLI / reqwest)'
);
