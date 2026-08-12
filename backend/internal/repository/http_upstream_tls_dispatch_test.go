package repository

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

// stubRoundTripper 测试用 RoundTripper：返回固定响应或错误并计数
type stubRoundTripper struct {
	calls atomic.Int64
	err   error
}

func (s *stubRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	s.calls.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
	}, nil
}

func newDispatchRequest(host string) *http.Request {
	return &http.Request{URL: mustParseDispatchURL("https://" + host + "/v1/test"), Header: make(http.Header)}
}

func mustParseDispatchURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

// TestTLSFingerprintDispatchPrefersH2 验证 h2 协商成功后按主机缓存，后续请求直接走 h2。
func TestTLSFingerprintDispatchPrefersH2(t *testing.T) {
	h1 := &stubRoundTripper{}
	h2 := &stubRoundTripper{}
	rt := &tlsFingerprintDispatchTransport{h1: h1, h2: h2}

	resp, err := rt.RoundTrip(newDispatchRequest("api.example.com"))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, int64(1), h2.calls.Load())
	require.Equal(t, int64(0), h1.calls.Load())

	// 第二个请求命中缓存，不再探测
	resp, err = rt.RoundTrip(newDispatchRequest("api.example.com"))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, int64(2), h2.calls.Load())
	require.Equal(t, int64(0), h1.calls.Load())
}

// TestTLSFingerprintDispatchFallsBackToH1 验证服务器不支持 h2 时回退 h1 并缓存。
func TestTLSFingerprintDispatchFallsBackToH1(t *testing.T) {
	h1 := &stubRoundTripper{}
	// 模拟 http2.Transport 对拨号错误的包装（%w 保留错误链）
	h2 := &stubRoundTripper{err: fmt.Errorf("http2: Transport dial: %w", tlsfingerprint.ErrH2NotNegotiated)}
	rt := &tlsFingerprintDispatchTransport{h1: h1, h2: h2}

	resp, err := rt.RoundTrip(newDispatchRequest("legacy.example.com"))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, int64(1), h2.calls.Load())
	require.Equal(t, int64(1), h1.calls.Load())

	// 回退结果被缓存，后续请求直接走 h1
	resp, err = rt.RoundTrip(newDispatchRequest("legacy.example.com"))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, int64(1), h2.calls.Load())
	require.Equal(t, int64(2), h1.calls.Load())
}

// TestTLSFingerprintDispatchPropagatesRealErrors 验证非 ALPN 错误直接透传且不缓存。
func TestTLSFingerprintDispatchPropagatesRealErrors(t *testing.T) {
	dialErr := errors.New("connection refused")
	h1 := &stubRoundTripper{}
	h2 := &stubRoundTripper{err: dialErr}
	rt := &tlsFingerprintDispatchTransport{h1: h1, h2: h2}

	_, err := rt.RoundTrip(newDispatchRequest("down.example.com"))
	require.ErrorIs(t, err, dialErr)
	require.Equal(t, int64(0), h1.calls.Load(), "真实错误不应回退 h1")

	// 未缓存：下次仍然先试 h2
	_, err = rt.RoundTrip(newDispatchRequest("down.example.com"))
	require.ErrorIs(t, err, dialErr)
	require.Equal(t, int64(2), h2.calls.Load())
}

// TestHasAcceptEncodingHeader 验证 Accept-Encoding 检测的大小写兼容。
func TestHasAcceptEncodingHeader(t *testing.T) {
	require.False(t, hasAcceptEncodingHeader(http.Header{}))
	require.True(t, hasAcceptEncodingHeader(http.Header{"Accept-Encoding": {"gzip"}}))
	require.True(t, hasAcceptEncodingHeader(http.Header{"accept-encoding": {"gzip, br"}}))
}
