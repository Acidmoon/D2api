// Package tlsfingerprint 模板构建与 ALPN 拨号的纯单元测试（无网络、无 build tag）。
package tlsfingerprint

import (
	"errors"
	"net/url"
	"sort"
	"testing"

	utls "github.com/refraction-networking/utls"
)

// specExtensionIDs 提取 spec 扩展的类型 ID 序列（用于断言顺序与构成）。
func specExtensionIDs(spec *utls.ClientHelloSpec) []uint16 {
	ids := make([]uint16, 0, len(spec.Extensions))
	for _, ext := range spec.Extensions {
		switch e := ext.(type) {
		case *utls.SNIExtension:
			ids = append(ids, 0)
		case *utls.StatusRequestExtension:
			ids = append(ids, 5)
		case *utls.SupportedCurvesExtension:
			ids = append(ids, 10)
		case *utls.SupportedPointsExtension:
			ids = append(ids, 11)
		case *utls.SignatureAlgorithmsExtension:
			ids = append(ids, 13)
		case *utls.ALPNExtension:
			ids = append(ids, 16)
		case *utls.SCTExtension:
			ids = append(ids, 18)
		case *utls.ExtendedMasterSecretExtension:
			ids = append(ids, 23)
		case *utls.SessionTicketExtension:
			ids = append(ids, 35)
		case *utls.SupportedVersionsExtension:
			ids = append(ids, 43)
		case *utls.PSKKeyExchangeModesExtension:
			ids = append(ids, 45)
		case *utls.SignatureAlgorithmsCertExtension:
			ids = append(ids, 50)
		case *utls.KeyShareExtension:
			ids = append(ids, 51)
		case *utls.RenegotiationInfoExtension:
			ids = append(ids, 0xff01)
		case *utls.UtlsGREASEExtension:
			ids = append(ids, 0x0a0a) // GREASE 占位
		case *utls.GREASEEncryptedClientHelloExtension:
			ids = append(ids, 65037)
		case *utls.GenericExtension:
			ids = append(ids, e.Id)
		default:
			ids = append(ids, 0xffff) // 未识别
		}
	}
	return ids
}

func findExtension[T utls.TLSExtension](spec *utls.ClientHelloSpec) T {
	var zero T
	for _, ext := range spec.Extensions {
		if e, ok := ext.(T); ok {
			return e
		}
	}
	return zero
}

// TestBuildClientHelloSpecGREASEComplete 验证 BoringSSL 系模板的完整 GREASE：
// cipher suites / supported_groups / supported_versions / key_share 首位均为 GREASE，
// 扩展首尾有 GREASE 书挡。
func TestBuildClientHelloSpecGREASEComplete(t *testing.T) {
	spec := buildClientHelloSpecFromProfile(DefaultNodeProfile())

	if spec.CipherSuites[0] != utls.GREASE_PLACEHOLDER {
		t.Errorf("cipher suites 首位应为 GREASE，实际 0x%04x", spec.CipherSuites[0])
	}
	curves := findExtension[*utls.SupportedCurvesExtension](spec)
	if curves == nil || curves.Curves[0] != utls.CurveID(utls.GREASE_PLACEHOLDER) {
		t.Errorf("supported_groups 首位应为 GREASE")
	}
	versions := findExtension[*utls.SupportedVersionsExtension](spec)
	if versions == nil || versions.Versions[0] != utls.GREASE_PLACEHOLDER {
		t.Errorf("supported_versions 首位应为 GREASE")
	}
	keyShare := findExtension[*utls.KeyShareExtension](spec)
	if keyShare == nil || keyShare.KeyShares[0].Group != utls.CurveID(utls.GREASE_PLACEHOLDER) {
		t.Errorf("key_share 首位应为 GREASE")
	}

	ids := specExtensionIDs(spec)
	if ids[0] != 0x0a0a || ids[len(ids)-1] != 0x0a0a {
		t.Errorf("扩展首尾应为 GREASE 书挡，实际 %v", ids)
	}
}

// TestBuildClientHelloSpecRustls 验证 rustls 模板：无 GREASE、SCSV 结尾、ALPN h2 优先。
func TestBuildClientHelloSpecRustls(t *testing.T) {
	spec := buildClientHelloSpecFromProfile(DefaultRustlsProfile())

	for i, cs := range spec.CipherSuites {
		if isGREASEValue(cs) {
			t.Errorf("rustls 模板不应包含 GREASE cipher（位置 %d）", i)
		}
	}
	if spec.CipherSuites[len(spec.CipherSuites)-1] != 0x00ff {
		t.Errorf("rustls cipher 列表末尾应为 SCSV(0x00ff)，实际 0x%04x", spec.CipherSuites[len(spec.CipherSuites)-1])
	}
	if spec.CipherSuites[0] != 0x1302 {
		t.Errorf("rustls 首选 cipher 应为 TLS_AES_256_GCM_SHA384(0x1302)，实际 0x%04x", spec.CipherSuites[0])
	}

	for _, id := range specExtensionIDs(spec) {
		if id == 0x0a0a {
			t.Error("rustls 模板不应包含 GREASE 扩展")
		}
	}

	alpn := findExtension[*utls.ALPNExtension](spec)
	if alpn == nil || len(alpn.AlpnProtocols) != 2 || alpn.AlpnProtocols[0] != "h2" {
		t.Errorf("rustls ALPN 应为 [h2 http/1.1]，实际 %v", alpn)
	}
}

// TestBuildClientHelloSpecDefaultALPN 验证默认 ALPN 已升级为 h2 优先。
func TestBuildClientHelloSpecDefaultALPN(t *testing.T) {
	spec := buildClientHelloSpecFromProfile(nil)
	alpn := findExtension[*utls.ALPNExtension](spec)
	if alpn == nil || alpn.AlpnProtocols[0] != "h2" || alpn.AlpnProtocols[1] != "http/1.1" {
		t.Fatalf("默认 ALPN 应为 [h2 http/1.1]，实际 %v", alpn.AlpnProtocols)
	}
}

// TestRustlsShuffleExtensions 验证 ShuffleExtensions 逐握手打乱扩展顺序，
// 且扩展集合保持不变（JA4 对顺序不敏感，JA3 不稳定正是 rustls 的真实行为）。
func TestRustlsShuffleExtensions(t *testing.T) {
	profile := DefaultRustlsProfile()
	reference := specExtensionIDs(buildClientHelloSpecFromProfile(profile))
	sortedReference := append([]uint16(nil), reference...)
	sort.Slice(sortedReference, func(i, j int) bool { return sortedReference[i] < sortedReference[j] })

	distinct := 0
	for i := 0; i < 64; i++ {
		ids := specExtensionIDs(buildClientHelloSpecFromProfile(profile))
		sorted := append([]uint16(nil), ids...)
		sort.Slice(sorted, func(a, b int) bool { return sorted[a] < sorted[b] })
		if len(sorted) != len(sortedReference) {
			t.Fatalf("洗牌后扩展数量变化: %d -> %d", len(sortedReference), len(sorted))
		}
		for j := range sorted {
			if sorted[j] != sortedReference[j] {
				t.Fatalf("洗牌后扩展集合变化: %v vs %v", sorted, sortedReference)
			}
		}
		same := true
		for j := range ids {
			if ids[j] != reference[j] {
				same = false
				break
			}
		}
		if !same {
			distinct++
		}
	}
	// 64 次全部与原序相同的概率约为 (1/11!)^63，实际不可能；仍只要求至少一次不同
	if distinct == 0 {
		t.Error("ShuffleExtensions 未生效：64 次构建扩展顺序全部相同")
	}
}

// TestCloneProfileWithALPN 验证 ALPN 覆盖副本不影响原 profile。
func TestCloneProfileWithALPN(t *testing.T) {
	orig := &Profile{Name: "p", ALPNProtocols: []string{"h2", "http/1.1"}, EnableGREASE: true}
	clone := cloneProfileWithALPN(orig, []string{"http/1.1"})

	if clone.ALPNProtocols[0] != "http/1.1" || len(clone.ALPNProtocols) != 1 {
		t.Errorf("clone ALPN 应为 [http/1.1]，实际 %v", clone.ALPNProtocols)
	}
	if orig.ALPNProtocols[0] != "h2" {
		t.Errorf("原 profile ALPN 被修改: %v", orig.ALPNProtocols)
	}
	if !clone.EnableGREASE || clone.Name != "p" {
		t.Errorf("clone 未保留其余字段: %+v", clone)
	}

	// nil profile 也能安全工作
	nilClone := cloneProfileWithALPN(nil, []string{"http/1.1"})
	if nilClone == nil || nilClone.ALPNProtocols[0] != "http/1.1" {
		t.Error("nil profile 的 clone 异常")
	}
}

// TestBuildDialTLSContextFuncs 验证按代理类型构建拨号函数的分支行为。
func TestBuildDialTLSContextFuncs(t *testing.T) {
	profile := &Profile{Name: "test"}

	// 直连
	h2Dial, h1Dial, err := BuildDialTLSContextFuncs(profile, nil)
	if err != nil || h2Dial == nil || h1Dial == nil {
		t.Fatalf("直连构建失败: %v", err)
	}

	// http / socks5 代理
	for _, raw := range []string{"http://proxy:8080", "socks5://proxy:1080", "socks5h://proxy:1080"} {
		u, _ := url.Parse(raw)
		if _, _, err := BuildDialTLSContextFuncs(profile, u); err != nil {
			t.Errorf("%s 构建失败: %v", raw, err)
		}
	}

	// https 代理 → ErrHTTPSProxyUnsupported
	u, _ := url.Parse("https://user:pass@proxy:8443")
	_, _, err = BuildDialTLSContextFuncs(profile, u)
	if !errors.Is(err, ErrHTTPSProxyUnsupported) {
		t.Errorf("https 代理应返回 ErrHTTPSProxyUnsupported，实际 %v", err)
	}

	// 未知协议 → 描述性错误
	u, _ = url.Parse("ftp://proxy:21")
	if _, _, err = BuildDialTLSContextFuncs(profile, u); err == nil {
		t.Error("未知代理协议应返回错误")
	}
}

// TestProfileSessionCache 验证会话缓存的模板隔离与 LRU 容量淘汰。
func TestProfileSessionCache(t *testing.T) {
	nodeCache := clientSessionCacheForProfile(&Profile{Name: "node"})
	rustlsCache := clientSessionCacheForProfile(&Profile{Name: "rustls"})

	state := &utls.ClientSessionState{}
	nodeCache.Put("host:443", state)

	if _, ok := nodeCache.Get("host:443"); !ok {
		t.Error("同模板应能取回 session")
	}
	if _, ok := rustlsCache.Get("host:443"); ok {
		t.Error("不同模板不应共享 session")
	}

	// Put nil 删除条目（接口约定）
	nodeCache.Put("host:443", nil)
	if _, ok := nodeCache.Get("host:443"); ok {
		t.Error("Put nil 应删除条目")
	}

	// LRU 容量淘汰
	cache := newLRUSessionCache(4)
	for _, k := range []string{"a", "b", "c", "d"} {
		cache.Put(k, state)
	}
	cache.Get("a") // a 变为最近使用
	cache.Put("e", state)
	if _, ok := cache.Get("b"); ok {
		t.Error("最久未使用的 b 应被淘汰")
	}
	for _, k := range []string{"a", "c", "d", "e"} {
		if _, ok := cache.Get(k); !ok {
			t.Errorf("%s 不应被淘汰", k)
		}
	}
}
