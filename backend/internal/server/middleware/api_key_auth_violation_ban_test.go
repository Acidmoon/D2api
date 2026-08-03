//go:build unit

package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubViolationBanCache struct {
	until   time.Time
	banned  bool
	getErr  error
	cleared int64
	set     int64
}

func (s *stubViolationBanCache) IncrementViolationCount(context.Context, int64, time.Duration) (int64, error) {
	return 0, nil
}
func (s *stubViolationBanCache) ResetViolationCount(context.Context, int64) error { return nil }
func (s *stubViolationBanCache) ClaimViolationNotifyCooldown(context.Context, int64, time.Duration) (bool, error) {
	return true, nil
}
func (s *stubViolationBanCache) SetUserViolationBan(_ context.Context, userID int64, _ time.Time, _ time.Duration) error {
	s.set = userID
	return nil
}
func (s *stubViolationBanCache) GetUserViolationBan(context.Context, int64) (time.Time, bool, error) {
	return s.until, s.banned, s.getErr
}
func (s *stubViolationBanCache) GetUserViolationBans(context.Context, []int64) (map[int64]time.Time, error) {
	return nil, s.getErr
}
func (s *stubViolationBanCache) ClaimViolationDedup(context.Context, int64, string, time.Duration) (bool, error) {
	return true, nil
}
func (s *stubViolationBanCache) ClearUserViolationBan(_ context.Context, userID int64) error {
	s.cleared = userID
	return nil
}

func newViolationBanAuthRouter(t *testing.T, cache service.ViolationCounterCache) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	user := &service.User{ID: 1001, Email: "u@example.test", Username: "alice", Status: service.StatusActive, Role: service.RoleUser}
	group := &service.Group{ID: 42, Name: "g", Status: service.StatusActive, Platform: service.PlatformAnthropic}
	repo := &stubApiKeyRepo{getByKey: func(context.Context, string) (*service.APIKey, error) {
		return &service.APIKey{
			ID: 7, UserID: user.ID, Key: "sk-test", Status: service.StatusActive,
			User: user, Group: group, GroupID: &group.ID,
		}, nil
	}}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	svc := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, cfg)
	svc.SetViolationBanCache(cache)
	r := gin.New()
	r.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(svc, nil, cfg)))
	r.GET("/v1/messages", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestAPIKeyAuth_UserViolationBanBlocksWith403(t *testing.T) {
	until := time.Now().Add(30 * time.Minute)
	r := newViolationBanAuthRouter(t, &stubViolationBanCache{until: until, banned: true})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("x-api-key", "sk-test")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "USER_VIOLATION_BANNED")
	require.Contains(t, w.Body.String(), "content policy violations")
}

func TestAPIKeyAuth_UserViolationBanAbsentPasses(t *testing.T) {
	r := newViolationBanAuthRouter(t, &stubViolationBanCache{banned: false})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("x-api-key", "sk-test")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAPIKeyAuth_UserViolationBanCacheErrorFailsOpen(t *testing.T) {
	r := newViolationBanAuthRouter(t, &stubViolationBanCache{getErr: errors.New("redis down")})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("x-api-key", "sk-test")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAPIKeyAuth_NoViolationBanCachePasses(t *testing.T) {
	r := newViolationBanAuthRouter(t, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
	req.Header.Set("x-api-key", "sk-test")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
