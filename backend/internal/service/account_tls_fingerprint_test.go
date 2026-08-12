package service

import "testing"

// 验证 TLS 指纹开关的平台/类型门槛：
// Anthropic OAuth/SetupToken 与 OpenAI OAuth 账号可启用，其余类型一律视为未启用。
func TestIsTLSFingerprintEnabled_PlatformGate(t *testing.T) {
	extra := map[string]any{"enable_tls_fingerprint": true}

	cases := []struct {
		name    string
		account Account
		want    bool
	}{
		{
			name:    "anthropic oauth enabled",
			account: Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth, Extra: extra},
			want:    true,
		},
		{
			name:    "anthropic setup token enabled",
			account: Account{Platform: PlatformAnthropic, Type: AccountTypeSetupToken, Extra: extra},
			want:    true,
		},
		{
			name:    "openai oauth enabled",
			account: Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: extra},
			want:    true,
		},
		{
			name:    "openai apikey not eligible",
			account: Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: extra},
			want:    false,
		},
		{
			name:    "gemini oauth not eligible",
			account: Account{Platform: PlatformGemini, Type: AccountTypeOAuth, Extra: extra},
			want:    false,
		},
		{
			name:    "enabled flag false",
			account: Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{"enable_tls_fingerprint": false}},
			want:    false,
		},
		{
			name:    "nil extra",
			account: Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth},
			want:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.account.IsTLSFingerprintEnabled(); got != tc.want {
				t.Fatalf("IsTLSFingerprintEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

// 验证平台感知的内置默认模板：
// OpenAI OAuth 账号未绑定模板时用 rustls（Codex CLI = reqwest+rustls），
// Anthropic 账号用 Node.js（Claude Code = BoringSSL），避免 UA 与 TLS 指纹跨层矛盾。
func TestResolveTLSProfile_PlatformDefault(t *testing.T) {
	svc := &TLSFingerprintProfileService{}
	enabled := map[string]any{"enable_tls_fingerprint": true}

	// OpenAI：rustls 模板（无 GREASE、ALPN h2 优先、cipher 末尾 SCSV）
	openai := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: enabled}
	p := svc.ResolveTLSProfile(openai)
	if p == nil {
		t.Fatal("OpenAI 账号应解析出内置模板")
	}
	if p.EnableGREASE {
		t.Error("rustls 模板不应启用 GREASE")
	}
	if len(p.ALPNProtocols) == 0 || p.ALPNProtocols[0] != "h2" {
		t.Errorf("rustls 模板 ALPN 应 h2 优先，实际 %v", p.ALPNProtocols)
	}
	if !p.ShuffleExtensions {
		t.Error("rustls 模板应启用扩展顺序随机化")
	}

	// Anthropic：Node.js 模板（完整 GREASE）
	anthropic := &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth, Extra: enabled}
	p = svc.ResolveTLSProfile(anthropic)
	if p == nil {
		t.Fatal("Anthropic 账号应解析出内置模板")
	}
	if !p.EnableGREASE {
		t.Error("Node.js 模板应启用 GREASE")
	}
	if p.ShuffleExtensions {
		t.Error("Node.js 模板不应随机化扩展顺序")
	}

	// 未启用指纹 → nil
	disabled := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	if got := svc.ResolveTLSProfile(disabled); got != nil {
		t.Errorf("未启用指纹应返回 nil，实际 %+v", got)
	}
}
