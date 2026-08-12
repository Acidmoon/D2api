// Package tlsfingerprint provides TLS fingerprint simulation for HTTP clients.
// It uses the utls library to create TLS connections that mimic real clients:
// Node.js 24.x (Claude Code, BoringSSL) or rustls (Codex CLI, reqwest+rustls).
package tlsfingerprint

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strings"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

// Profile contains TLS fingerprint configuration.
// All slice fields use built-in defaults when empty.
type Profile struct {
	Name                string // Profile name for identification
	CipherSuites        []uint16
	Curves              []uint16
	PointFormats        []uint16
	EnableGREASE        bool
	SignatureAlgorithms []uint16 // Empty uses defaultSignatureAlgorithms
	ALPNProtocols       []string // Empty uses ["h2", "http/1.1"]
	SupportedVersions   []uint16 // Empty uses [TLS1.3, TLS1.2]
	KeyShareGroups      []uint16 // Empty uses [X25519]
	PSKModes            []uint16 // Empty uses [psk_dhe_ke]
	Extensions          []uint16 // Extension type IDs in order; empty uses default Node.js 24.x order
	// ShuffleExtensions 为 true 时每次握手随机打乱扩展顺序（rustls 0.23.21+ 的反指纹行为）。
	// 仅运行时字段，不入库；DB 模板中的 rustls 模板使用固定构造顺序。
	ShuffleExtensions bool
}

// Dialer creates TLS connections with custom fingerprints.
type Dialer struct {
	profile    *Profile
	baseDialer func(ctx context.Context, network, addr string) (net.Conn, error)
}

// HTTPProxyDialer creates TLS connections through HTTP/HTTPS proxies with custom fingerprints.
// It handles the CONNECT tunnel establishment before performing TLS handshake.
type HTTPProxyDialer struct {
	profile  *Profile
	proxyURL *url.URL
}

// SOCKS5ProxyDialer creates TLS connections through SOCKS5 proxies with custom fingerprints.
// It uses golang.org/x/net/proxy to establish the SOCKS5 tunnel.
type SOCKS5ProxyDialer struct {
	profile  *Profile
	proxyURL *url.URL
}

// Default TLS fingerprint values captured from Claude Code (Node.js 24.x)
// Captured via tls-fingerprint-web capture server
// JA3 Hash: 44f88fca027f27bab4bb08d4af15f23e
// JA4:      t13d1714h1_5b57614c22b0_7baf387fc6ff
var (
	// defaultCipherSuites contains the 17 cipher suites from Node.js 24.x
	// Order is critical for JA3 fingerprint matching
	defaultCipherSuites = []uint16{
		// TLS 1.3 cipher suites
		0x1301, // TLS_AES_128_GCM_SHA256
		0x1302, // TLS_AES_256_GCM_SHA384
		0x1303, // TLS_CHACHA20_POLY1305_SHA256

		// ECDHE + AES-GCM
		0xc02b, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
		0xc02f, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
		0xc02c, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
		0xc030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384

		// ECDHE + ChaCha20-Poly1305
		0xcca9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
		0xcca8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256

		// ECDHE + AES-CBC-SHA (legacy fallback)
		0xc009, // TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
		0xc013, // TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
		0xc00a, // TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
		0xc014, // TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA

		// RSA + AES-GCM (non-PFS)
		0x009c, // TLS_RSA_WITH_AES_128_GCM_SHA256
		0x009d, // TLS_RSA_WITH_AES_256_GCM_SHA384

		// RSA + AES-CBC-SHA (non-PFS, legacy)
		0x002f, // TLS_RSA_WITH_AES_128_CBC_SHA
		0x0035, // TLS_RSA_WITH_AES_256_CBC_SHA
	}

	// defaultCurves contains the 3 supported groups from Node.js 24.x
	defaultCurves = []utls.CurveID{
		utls.X25519,    // 0x001d
		utls.CurveP256, // 0x0017 (secp256r1)
		utls.CurveP384, // 0x0018 (secp384r1)
	}

	// defaultPointFormats contains point formats from Node.js 24.x
	defaultPointFormats = []uint16{
		0, // uncompressed
	}

	// defaultSignatureAlgorithms contains the 9 signature algorithms from Node.js 24.x
	defaultSignatureAlgorithms = []utls.SignatureScheme{
		0x0403, // ecdsa_secp256r1_sha256
		0x0804, // rsa_pss_rsae_sha256
		0x0401, // rsa_pkcs1_sha256
		0x0503, // ecdsa_secp384r1_sha384
		0x0805, // rsa_pss_rsae_sha384
		0x0501, // rsa_pkcs1_sha384
		0x0806, // rsa_pss_rsae_sha512
		0x0601, // rsa_pkcs1_sha512
		0x0201, // rsa_pkcs1_sha1
	}

	// defaultALPNProtocols 默认 ALPN 列表：h2 优先、http/1.1 兜底。
	// 真实 Node.js（undici）与 reqwest 都声明 h2+http/1.1；此前仅声明 http/1.1
	// 是 Go 反代的显著特征，且与 Codex UA（声称支持 h2 的 reqwest）形成跨层矛盾。
	defaultALPNProtocols = []string{"h2", "http/1.1"}
)

// rustls（Codex CLI / codex-rs，reqwest + rustls ring provider）默认 ClientHello 特征。
// 依据：
//   - rustls 0.23 源码 rustls/src/client/hs.rs（emit_client_hello_for_retry 的扩展构造顺序）
//   - rustls crypto/ring provider 的默认 cipher suites / kx groups / verify schemes 顺序
//   - rustls 不发送任何 GREASE 值；TLS1.2 安全重协商以 SCSV（0x00ff）附加在
//     cipher 列表末尾，而非使用 renegotiation_info 扩展
//   - rustls 0.23.21+ 会在发送前用随机种子重排扩展顺序（内置模板用 ShuffleExtensions 复现）
var (
	// rustlsCipherSuites rustls ring provider 默认顺序 + 末尾 SCSV
	rustlsCipherSuites = []uint16{
		0x1302, // TLS_AES_256_GCM_SHA384
		0x1303, // TLS_CHACHA20_POLY1305_SHA256
		0x1301, // TLS_AES_128_GCM_SHA256
		0xc02b, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
		0xc02f, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
		0xc02c, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
		0xc030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
		0xcca9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
		0xcca8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
		0x00ff, // TLS_EMPTY_RENEGOTIATION_INFO_SCSV（rustls 固定附加在末尾）
	}

	// rustlsCurves ring provider 默认 kx groups：X25519, secp256r1, secp384r1
	rustlsCurves = []uint16{0x001d, 0x0017, 0x0018}

	// rustlsSignatureAlgorithms webpki verifier 默认 supported_verify_schemes 顺序
	rustlsSignatureAlgorithms = []uint16{
		0x0403, // ecdsa_secp256r1_sha256
		0x0503, // ecdsa_secp384r1_sha384
		0x0603, // ecdsa_secp521r1_sha512
		0x0807, // ed25519
		0x0806, // rsa_pss_rsae_sha512
		0x0805, // rsa_pss_rsae_sha384
		0x0804, // rsa_pss_rsae_sha256
		0x0601, // rsa_pkcs1_sha512
		0x0501, // rsa_pkcs1_sha384
		0x0401, // rsa_pkcs1_sha256
	}

	// rustlsExtensionOrder rustls 的扩展构造顺序（发送前会被 rustls 随机重排）。
	// 依次来自 hs.rs：supported_versions → supported_groups → signature_algorithms →
	// extended_master_secret → status_request(OCSP) → ec_point_formats → server_name →
	// key_share → psk_key_exchange_modes → alpn → session_ticket（无会话可恢复时请求新票据）。
	rustlsExtensionOrder = []uint16{
		43, // supported_versions
		10, // supported_groups
		13, // signature_algorithms
		23, // extended_master_secret
		5,  // status_request (OCSP)
		11, // ec_point_formats
		0,  // server_name
		51, // key_share
		45, // psk_key_exchange_modes
		16, // alpn
		35, // session_ticket
	}
)

// DefaultNodeProfile 返回内置 Node.js 24.x（Claude Code）指纹模板。
// 启用完整 GREASE（BoringSSL 在 cipher/groups/versions/key_share 均插入 GREASE 值），
// ALPN 声明 h2+http/1.1，其余字段留空走内置默认（Node.js 24.x 抓包值）。
func DefaultNodeProfile() *Profile {
	return &Profile{Name: "Built-in Node.js 24.x (Claude Code)", EnableGREASE: true}
}

// DefaultRustlsProfile 返回内置 rustls（Codex CLI，reqwest+rustls）指纹模板。
// rustls 不使用 GREASE，且每次握手随机化扩展顺序（ShuffleExtensions）。
func DefaultRustlsProfile() *Profile {
	return &Profile{
		Name:                "Built-in rustls (Codex CLI)",
		EnableGREASE:        false,
		CipherSuites:        append([]uint16(nil), rustlsCipherSuites...),
		Curves:              append([]uint16(nil), rustlsCurves...),
		PointFormats:        []uint16{0},
		SignatureAlgorithms: append([]uint16(nil), rustlsSignatureAlgorithms...),
		ALPNProtocols:       append([]string(nil), defaultALPNProtocols...),
		SupportedVersions:   []uint16{utls.VersionTLS13, utls.VersionTLS12},
		KeyShareGroups:      []uint16{uint16(utls.X25519)},
		PSKModes:            []uint16{uint16(utls.PskModeDHE)},
		Extensions:          append([]uint16(nil), rustlsExtensionOrder...),
		ShuffleExtensions:   true,
	}
}

// NewDialer creates a new TLS fingerprint dialer.
// baseDialer is used for TCP connection establishment (supports proxy scenarios).
// If baseDialer is nil, direct TCP dial is used.
func NewDialer(profile *Profile, baseDialer func(ctx context.Context, network, addr string) (net.Conn, error)) *Dialer {
	if baseDialer == nil {
		baseDialer = (&net.Dialer{}).DialContext
	}
	return &Dialer{profile: profile, baseDialer: baseDialer}
}

// NewHTTPProxyDialer creates a new TLS fingerprint dialer that works through HTTP/HTTPS proxies.
// It establishes a CONNECT tunnel before performing TLS handshake with custom fingerprint.
func NewHTTPProxyDialer(profile *Profile, proxyURL *url.URL) *HTTPProxyDialer {
	return &HTTPProxyDialer{profile: profile, proxyURL: proxyURL}
}

// NewSOCKS5ProxyDialer creates a new TLS fingerprint dialer that works through SOCKS5 proxies.
// It establishes a SOCKS5 tunnel before performing TLS handshake with custom fingerprint.
func NewSOCKS5ProxyDialer(profile *Profile, proxyURL *url.URL) *SOCKS5ProxyDialer {
	return &SOCKS5ProxyDialer{profile: profile, proxyURL: proxyURL}
}

// DialTLSContext establishes a TLS connection through SOCKS5 proxy with the configured fingerprint.
// Flow: SOCKS5 CONNECT to target -> TLS handshake with utls on the tunnel
func (d *SOCKS5ProxyDialer) DialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	slog.Debug("tls_fingerprint_socks5_connecting", "proxy", d.proxyURL.Host, "target", addr)

	// Step 1: Create SOCKS5 dialer
	var auth *proxy.Auth
	if d.proxyURL.User != nil {
		username := d.proxyURL.User.Username()
		password, _ := d.proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     username,
			Password: password,
		}
	}

	// Determine proxy address
	proxyAddr := d.proxyURL.Host
	if d.proxyURL.Port() == "" {
		proxyAddr = net.JoinHostPort(d.proxyURL.Hostname(), "1080") // Default SOCKS5 port
	}

	socksDialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
	if err != nil {
		slog.Debug("tls_fingerprint_socks5_dialer_failed", "error", err)
		return nil, fmt.Errorf("create SOCKS5 dialer: %w", err)
	}

	// Step 2: Establish SOCKS5 tunnel to target
	slog.Debug("tls_fingerprint_socks5_establishing_tunnel", "target", addr)
	conn, err := socksDialer.Dial("tcp", addr)
	if err != nil {
		slog.Debug("tls_fingerprint_socks5_connect_failed", "error", err)
		return nil, fmt.Errorf("SOCKS5 connect: %w", err)
	}
	slog.Debug("tls_fingerprint_socks5_tunnel_established")

	// Step 3: Perform TLS handshake on the tunnel with utls fingerprint
	return performTLSHandshake(ctx, conn, d.profile, addr)
}

// DialTLSContext establishes a TLS connection through HTTP proxy with the configured fingerprint.
// Flow: TCP connect to proxy -> CONNECT tunnel -> TLS handshake with utls
func (d *HTTPProxyDialer) DialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	slog.Debug("tls_fingerprint_http_proxy_connecting", "proxy", d.proxyURL.Host, "target", addr)

	// Step 1: TCP connect to proxy server
	var proxyAddr string
	if d.proxyURL.Port() != "" {
		proxyAddr = d.proxyURL.Host
	} else {
		// Default ports
		if d.proxyURL.Scheme == "https" {
			proxyAddr = net.JoinHostPort(d.proxyURL.Hostname(), "443")
		} else {
			proxyAddr = net.JoinHostPort(d.proxyURL.Hostname(), "80")
		}
	}

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		slog.Debug("tls_fingerprint_http_proxy_connect_failed", "error", err)
		return nil, fmt.Errorf("connect to proxy: %w", err)
	}
	slog.Debug("tls_fingerprint_http_proxy_connected", "proxy_addr", proxyAddr)

	// Step 2: Send CONNECT request to establish tunnel
	req := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}

	// Add proxy authentication if present
	if d.proxyURL.User != nil {
		username := d.proxyURL.User.Username()
		password, _ := d.proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Proxy-Authorization", "Basic "+auth)
	}

	slog.Debug("tls_fingerprint_http_proxy_sending_connect", "target", addr)
	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		slog.Debug("tls_fingerprint_http_proxy_write_failed", "error", err)
		return nil, fmt.Errorf("write CONNECT request: %w", err)
	}

	// Step 3: Read CONNECT response
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		_ = conn.Close()
		slog.Debug("tls_fingerprint_http_proxy_read_response_failed", "error", err)
		return nil, fmt.Errorf("read CONNECT response: %w", err)
	}
	// CONNECT response has no body; do not defer resp.Body.Close() as it wraps the
	// same conn that will be used for the TLS handshake.

	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		slog.Debug("tls_fingerprint_http_proxy_connect_failed_status", "status_code", resp.StatusCode, "status", resp.Status)
		return nil, fmt.Errorf("proxy CONNECT failed: %s", resp.Status)
	}
	slog.Debug("tls_fingerprint_http_proxy_tunnel_established")

	// Step 4: Perform TLS handshake on the tunnel with utls fingerprint
	return performTLSHandshake(ctx, conn, d.profile, addr)
}

// DialTLSContext establishes a TLS connection with the configured fingerprint.
// This method is designed to be used as http.Transport.DialTLSContext.
func (d *Dialer) DialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Establish TCP connection using base dialer (supports proxy)
	slog.Debug("tls_fingerprint_dialing_tcp", "addr", addr)
	conn, err := d.baseDialer(ctx, network, addr)
	if err != nil {
		slog.Debug("tls_fingerprint_tcp_dial_failed", "error", err)
		return nil, err
	}
	slog.Debug("tls_fingerprint_tcp_connected", "addr", addr)

	// Perform TLS handshake with utls fingerprint
	return performTLSHandshake(ctx, conn, d.profile, addr)
}

// performTLSHandshake performs the uTLS handshake on an established connection.
// It builds a ClientHello spec from the profile, applies it, and completes the handshake.
// On failure, conn is closed and an error is returned.
func performTLSHandshake(ctx context.Context, conn net.Conn, profile *Profile, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	spec := buildClientHelloSpecFromProfile(profile)
	tlsConn := utls.UClient(conn, &utls.Config{
		ServerName: host,
		// 启用 TLS session resumption：真实客户端（Node/rustls）都会复用会话票据，
		// 每次都做完整握手反而是自动化客户端特征。缓存按模板名隔离（见 session_cache.go）。
		ClientSessionCache: clientSessionCacheForProfile(profile),
	}, utls.HelloCustom)

	if err := tlsConn.ApplyPreset(spec); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("apply TLS preset: %w", err)
	}

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	state := tlsConn.ConnectionState()
	slog.Debug("tls_fingerprint_handshake_success",
		"host", host,
		"version", state.Version,
		"cipher_suite", state.CipherSuite,
		"alpn", state.NegotiatedProtocol)

	return tlsConn, nil
}

// toUTLSCurves converts uint16 slice to utls.CurveID slice.
func toUTLSCurves(curves []uint16) []utls.CurveID {
	result := make([]utls.CurveID, len(curves))
	for i, c := range curves {
		result[i] = utls.CurveID(c)
	}
	return result
}

// defaultExtensionOrder is the Node.js 24.x extension order.
// Used when Profile.Extensions is empty.
var defaultExtensionOrder = []uint16{
	0,     // server_name
	65037, // encrypted_client_hello
	23,    // extended_master_secret
	65281, // renegotiation_info
	10,    // supported_groups
	11,    // ec_point_formats
	35,    // session_ticket
	16,    // alpn
	5,     // status_request
	13,    // signature_algorithms
	18,    // signed_certificate_timestamp
	51,    // key_share
	45,    // psk_key_exchange_modes
	43,    // supported_versions
}

// isGREASEValue checks if a uint16 value matches the TLS GREASE pattern (0x?a?a).
func isGREASEValue(v uint16) bool {
	return v&0x0f0f == 0x0a0a && v>>8 == v&0xff
}

// buildClientHelloSpecFromProfile constructs ClientHelloSpec from a Profile.
// This is a standalone function that can be used by both Dialer and HTTPProxyDialer.
func buildClientHelloSpecFromProfile(profile *Profile) *utls.ClientHelloSpec {
	// Resolve effective values (profile overrides or built-in defaults)
	cipherSuites := defaultCipherSuites
	if profile != nil && len(profile.CipherSuites) > 0 {
		cipherSuites = profile.CipherSuites
	}

	curves := defaultCurves
	if profile != nil && len(profile.Curves) > 0 {
		curves = toUTLSCurves(profile.Curves)
	}

	pointFormats := defaultPointFormats
	if profile != nil && len(profile.PointFormats) > 0 {
		pointFormats = profile.PointFormats
	}

	signatureAlgorithms := defaultSignatureAlgorithms
	if profile != nil && len(profile.SignatureAlgorithms) > 0 {
		signatureAlgorithms = make([]utls.SignatureScheme, len(profile.SignatureAlgorithms))
		for i, s := range profile.SignatureAlgorithms {
			signatureAlgorithms[i] = utls.SignatureScheme(s)
		}
	}

	alpnProtocols := defaultALPNProtocols
	if profile != nil && len(profile.ALPNProtocols) > 0 {
		alpnProtocols = profile.ALPNProtocols
	}

	supportedVersions := []uint16{utls.VersionTLS13, utls.VersionTLS12}
	if profile != nil && len(profile.SupportedVersions) > 0 {
		supportedVersions = profile.SupportedVersions
	}

	keyShareGroups := []utls.CurveID{utls.X25519}
	if profile != nil && len(profile.KeyShareGroups) > 0 {
		keyShareGroups = toUTLSCurves(profile.KeyShareGroups)
	}

	pskModes := []uint16{uint16(utls.PskModeDHE)}
	if profile != nil && len(profile.PSKModes) > 0 {
		pskModes = profile.PSKModes
	}

	enableGREASE := profile != nil && profile.EnableGREASE

	// GREASE 完整化：真实 BoringSSL 不仅在扩展首尾插 GREASE，还会在 cipher suites、
	// supported_groups、supported_versions、key_share 的首位各插一个 GREASE 值。
	// uTLS ApplyPreset 会把 GREASE_PLACEHOLDER（0x0a0a）替换为按 BoringSSL 算法
	// 派生的真实 GREASE 值。rustls 不发送 GREASE，因此由 Profile.EnableGREASE 控制，
	// 仅 BoringSSL 系（Node.js）模板开启。
	if enableGREASE {
		cipherSuites = append([]uint16{utls.GREASE_PLACEHOLDER}, cipherSuites...)
		curves = append([]utls.CurveID{utls.CurveID(utls.GREASE_PLACEHOLDER)}, curves...)
		supportedVersions = append([]uint16{utls.GREASE_PLACEHOLDER}, supportedVersions...)
	}

	// Build key shares
	keyShares := make([]utls.KeyShare, len(keyShareGroups))
	for i, g := range keyShareGroups {
		keyShares[i] = utls.KeyShare{Group: g}
	}
	if enableGREASE {
		// key_share 首位的 GREASE 占位组带 1 字节空数据（BoringSSL 行为）
		keyShares = append([]utls.KeyShare{{Group: utls.CurveID(utls.GREASE_PLACEHOLDER), Data: []byte{0}}}, keyShares...)
	}

	// Determine extension order
	extOrder := defaultExtensionOrder
	if profile != nil && len(profile.Extensions) > 0 {
		extOrder = profile.Extensions
	}

	// Build extensions list from the ordered IDs.
	// Parametric extensions (curves, sigalgs, etc.) are populated with resolved profile values.
	// Unknown IDs use GenericExtension (sends type ID with empty data).
	extensions := make([]utls.TLSExtension, 0, len(extOrder)+2)
	for _, id := range extOrder {
		if isGREASEValue(id) {
			extensions = append(extensions, &utls.UtlsGREASEExtension{})
			continue
		}
		switch id {
		case 0: // server_name
			extensions = append(extensions, &utls.SNIExtension{})
		case 5: // status_request (OCSP)
			extensions = append(extensions, &utls.StatusRequestExtension{})
		case 10: // supported_groups
			extensions = append(extensions, &utls.SupportedCurvesExtension{Curves: curves})
		case 11: // ec_point_formats
			extensions = append(extensions, &utls.SupportedPointsExtension{SupportedPoints: toUint8s(pointFormats)})
		case 13: // signature_algorithms
			extensions = append(extensions, &utls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: signatureAlgorithms})
		case 16: // alpn
			extensions = append(extensions, &utls.ALPNExtension{AlpnProtocols: alpnProtocols})
		case 18: // signed_certificate_timestamp
			extensions = append(extensions, &utls.SCTExtension{})
		case 23: // extended_master_secret
			extensions = append(extensions, &utls.ExtendedMasterSecretExtension{})
		case 35: // session_ticket
			extensions = append(extensions, &utls.SessionTicketExtension{})
		case 43: // supported_versions
			extensions = append(extensions, &utls.SupportedVersionsExtension{Versions: supportedVersions})
		case 45: // psk_key_exchange_modes
			extensions = append(extensions, &utls.PSKKeyExchangeModesExtension{Modes: toUint8s(pskModes)})
		case 50: // signature_algorithms_cert
			extensions = append(extensions, &utls.SignatureAlgorithmsCertExtension{SupportedSignatureAlgorithms: signatureAlgorithms})
		case 51: // key_share
			extensions = append(extensions, &utls.KeyShareExtension{KeyShares: keyShares})
		case 0xfe0d: // encrypted_client_hello (ECH, 65037)
			// Send GREASE ECH with random payload — mimics Node.js behavior when no real ECHConfig is available.
			// An empty GenericExtension causes "error decoding message" from servers that validate ECH format.
			extensions = append(extensions, &utls.GREASEEncryptedClientHelloExtension{})
		case 0xff01: // renegotiation_info
			extensions = append(extensions, &utls.RenegotiationInfoExtension{})
		default:
			// Unknown extension — send as GenericExtension (type ID + empty data).
			// This covers encrypt_then_mac(22) and any future extensions.
			extensions = append(extensions, &utls.GenericExtension{Id: id})
		}
	}

	// For default extension order with EnableGREASE, wrap with GREASE bookends
	if enableGREASE && (profile == nil || len(profile.Extensions) == 0) {
		extensions = append([]utls.TLSExtension{&utls.UtlsGREASEExtension{}}, extensions...)
		extensions = append(extensions, &utls.UtlsGREASEExtension{})
	}

	// rustls 0.23.21+ 行为：发送前按随机种子重排扩展顺序（PSK 恒在最后，
	// 本实现不静态声明 pre_shared_key，故整体打乱即可）。
	// 仅 ShuffleExtensions 模板启用；BoringSSL 系模板保持固定顺序。
	if profile != nil && profile.ShuffleExtensions {
		rand.Shuffle(len(extensions), func(i, j int) {
			extensions[i], extensions[j] = extensions[j], extensions[i]
		})
	}

	return &utls.ClientHelloSpec{
		CipherSuites:       cipherSuites,
		CompressionMethods: []uint8{0}, // null compression only (standard)
		Extensions:         extensions,
		TLSVersMax:         utls.VersionTLS13,
		TLSVersMin:         utls.VersionTLS10,
	}
}

// toUint8s converts []uint16 to []uint8 (for utls fields that require []uint8).
func toUint8s(vals []uint16) []uint8 {
	out := make([]uint8, len(vals))
	for i, v := range vals {
		out[i] = uint8(v)
	}
	return out
}

// ErrH2NotNegotiated 表示 TLS 握手成功但 ALPN 未协商到 h2。
// 调用方（协议分发 transport）据此回退到 HTTP/1.1。
var ErrH2NotNegotiated = errors.New("tlsfingerprint: ALPN 未协商到 h2")

// ErrHTTPSProxyUnsupported 表示 https:// 代理无法承载指纹握手：
// CONNECT 隧道前需要先对代理本身做 TLS，当前指纹 dialer 只支持明文 CONNECT。
var ErrHTTPSProxyUnsupported = errors.New("tlsfingerprint: https 代理不支持 TLS 指纹握手")

// BuildDialTLSContextFuncs 按代理类型构建两套 DialTLSContext 函数：
//   - h2Dial：按 profile 声明的 ALPN（通常 h2+http/1.1）握手；协商结果不是 h2 时
//     关闭连接并返回包装了 ErrH2NotNegotiated 的错误，供调用方回退 h1；
//   - h1Dial：ALPN 强制收敛为 ["http/1.1"]，用于明确的 h1 路径，避免服务器
//     协商出 h2 而 http.Transport 按 HTTP/1.1 说话的协议错配。
//
// 代理处理与 buildUpstreamTransportWithTLSFingerprint 的历史行为一致：
// nil=直连，http=CONNECT 隧道，socks5/socks5h=SOCKS5 隧道；
// https 代理返回 ErrHTTPSProxyUnsupported，未知协议返回描述性错误。
func BuildDialTLSContextFuncs(profile *Profile, proxyURL *url.URL) (h2Dial, h1Dial func(ctx context.Context, network, addr string) (net.Conn, error), err error) {
	base, err := baseDialTLSContext(profile, proxyURL)
	if err != nil {
		return nil, nil, err
	}
	h1Base, err := baseDialTLSContext(cloneProfileWithALPN(profile, []string{"http/1.1"}), proxyURL)
	if err != nil {
		return nil, nil, err
	}

	h2Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := base(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		if uconn, ok := conn.(*utls.UConn); ok {
			if proto := uconn.ConnectionState().NegotiatedProtocol; proto != "h2" {
				_ = conn.Close()
				return nil, fmt.Errorf("%w（协商结果 %q）", ErrH2NotNegotiated, proto)
			}
		}
		return conn, nil
	}
	return h2Dial, h1Base, nil
}

// baseDialTLSContext 按代理类型返回底层 DialTLSContext 函数
func baseDialTLSContext(profile *Profile, proxyURL *url.URL) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	if proxyURL == nil {
		return NewDialer(profile, nil).DialTLSContext, nil
	}
	switch strings.ToLower(proxyURL.Scheme) {
	case "http":
		return NewHTTPProxyDialer(profile, proxyURL).DialTLSContext, nil
	case "socks5", "socks5h":
		return NewSOCKS5ProxyDialer(profile, proxyURL).DialTLSContext, nil
	case "https":
		return nil, ErrHTTPSProxyUnsupported
	default:
		return nil, fmt.Errorf("tlsfingerprint: 未知代理协议 %q", proxyURL.Scheme)
	}
}

// cloneProfileWithALPN 复制 profile 并覆盖 ALPN 声明。
// 浅拷贝：其余切片字段只读共享，不会被修改。
func cloneProfileWithALPN(profile *Profile, alpn []string) *Profile {
	clone := &Profile{}
	if profile != nil {
		*clone = *profile
	}
	clone.ALPNProtocols = alpn
	return clone
}
