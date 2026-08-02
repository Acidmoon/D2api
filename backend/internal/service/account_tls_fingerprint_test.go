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
