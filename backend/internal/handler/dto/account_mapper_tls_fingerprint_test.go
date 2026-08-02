package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 回归：OpenAI OAuth 账号启用 TLS 指纹后，DTO 必须暴露开关与模板 ID，
// 否则前端编辑弹窗回读不到（保存成功但重新打开显示未启用）。
func TestAccountFromServiceShallow_TLSFingerprintFields(t *testing.T) {
	newAccount := func(platform, typ string) *service.Account {
		return &service.Account{
			ID:       1,
			Platform: platform,
			Type:     typ,
			Extra: map[string]any{
				"enable_tls_fingerprint":     true,
				"tls_fingerprint_profile_id": float64(7),
			},
		}
	}

	cases := []struct {
		name        string
		platform    string
		typ         string
		wantEnabled bool
	}{
		{"anthropic oauth", service.PlatformAnthropic, service.AccountTypeOAuth, true},
		{"anthropic setup token", service.PlatformAnthropic, service.AccountTypeSetupToken, true},
		{"openai oauth", service.PlatformOpenAI, service.AccountTypeOAuth, true},
		{"openai apikey not eligible", service.PlatformOpenAI, service.AccountTypeAPIKey, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := AccountFromServiceShallow(newAccount(tc.platform, tc.typ))
			if tc.wantEnabled {
				if out.EnableTLSFingerprint == nil || !*out.EnableTLSFingerprint {
					t.Fatalf("EnableTLSFingerprint = %v, want true", out.EnableTLSFingerprint)
				}
				if out.TLSFingerprintProfileID == nil || *out.TLSFingerprintProfileID != 7 {
					t.Fatalf("TLSFingerprintProfileID = %v, want 7", out.TLSFingerprintProfileID)
				}
			} else {
				if out.EnableTLSFingerprint != nil {
					t.Fatalf("EnableTLSFingerprint = %v, want nil", *out.EnableTLSFingerprint)
				}
			}
		})
	}
}
