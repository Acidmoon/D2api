# tlsfingerprint — TLS 指纹模拟

基于 [uTLS](https://github.com/refraction-networking/utls) 的自定义 ClientHelloSpec，
让出站请求的 TLS 握手特征与账号宣称的客户端身份一致，避免被上游风控识别为反代。

## 模板体系

| 模板 | 模拟对象 | 特点 |
| --- | --- | --- |
| `DefaultNodeProfile()` | Node.js 24.x（Claude Code，BoringSSL） | 完整 GREASE（cipher/groups/versions/key_share/扩展书挡）、固定扩展顺序 |
| `DefaultRustlsProfile()` | rustls 0.23（Codex CLI，reqwest+rustls ring provider） | 无 GREASE、cipher 末尾 SCSV(0x00ff)、`ShuffleExtensions` 逐握手随机扩展顺序 |
| DB 模板 | 管理界面自定义 | 完整 ClientHello 参数入库（`tls_fingerprint_profiles` 表） |

- 账号侧开关：`Account.Extra["enable_tls_fingerprint"]` + `tls_fingerprint_profile_id`
  （0=内置默认、-1=随机、>0=指定 DB 模板）。
- **平台感知默认**：未绑定模板时由 `service.ResolveTLSProfile` 按平台选择——
  OpenAI 账号用 rustls 模板（UA 是 codex-tui / Rust），Anthropic 账号用 Node.js 模板
  （Claude Code 是 Node.js）。避免 UA/originator 与 TLS 指纹的跨层矛盾。
- DB 中的 rustls 种子模板（迁移 222）存固定扩展构造顺序；代码内置 rustls 模板
  额外启用 `ShuffleExtensions`（`Profile.ShuffleExtensions` 仅运行时，不入库）。

## 关键行为

- **ALPN / HTTP2**：默认 ALPN 为 `["h2", "http/1.1"]`。`BuildDialTLSContextFuncs`
  返回两套拨号函数：h2 路径校验协商结果（非 h2 返回 `ErrH2NotNegotiated`），
  h1 路径把 ALPN 收敛为仅 `http/1.1`（防协议错配）。分发逻辑见
  `repository/http_upstream.go` 的 `tlsFingerprintDispatchTransport`（按主机缓存协商结果）。
- **Session resumption**：`performTLSHandshake` 挂接 `ClientSessionCache`
  （`session_cache.go`：进程级 LRU，容量 1024，按模板名前缀隔离）。
  uTLS 的 sessionController 会自动处理 SessionTicket / pre_shared_key 的注入与消费。
- **GREASE**：仅 `EnableGREASE=true` 的 BoringSSL 系模板插入；uTLS ApplyPreset
  会把 `GREASE_PLACEHOLDER` 替换为按 BoringSSL 算法派生的真实 GREASE 值。
  rustls 模板严禁 GREASE。
- **https:// 代理**：指纹 dialer 只支持明文 CONNECT，https 代理回退为无指纹
  transport 并打 Warn 日志（`tls_fingerprint_https_proxy_fallback`）。

## 已知限制

- h2 指纹：x/net/http2 的 SETTINGS 只暴露有限旋钮（MaxReadFrameSize/HeaderTableSize
  等默认值已与常见客户端一致）；`INITIAL_WINDOW_SIZE`（4MB）与连接级
  `WINDOW_UPDATE`（约 1GB）无法通过公开 API 调整，仍保留 Go 特有值。
- h2 改造的连接复用语义由 `http2.Transport` 自管（单连接多路复用），
  `MaxIdleConnsPerHost`/`MaxConnsPerHost` 等 h1 连接池参数只对 h1 分支生效。

## 测试

```bash
# 纯单元测试（spec 构建 / GREASE / 洗牌 / 会话缓存 / 拨号函数分支）
go test ./internal/pkg/tlsfingerprint/ -count=1

# 网络验证（需要外网，比对 tls.peet.ws 指纹）
TLSFINGERPRINT_NETWORK_TESTS=1 go test -v -tags=unit ./internal/pkg/tlsfingerprint/
```
