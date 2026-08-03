package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fakeViolationBanCache 是 service.ViolationCounterCache 的内存实现，
// 供封禁可见性/手动封禁的 handler 测试使用。
type fakeViolationBanCache struct {
	bans   map[int64]time.Time
	getErr error
	sets   map[int64]time.Duration
}

func newFakeViolationBanCache() *fakeViolationBanCache {
	return &fakeViolationBanCache{bans: map[int64]time.Time{}, sets: map[int64]time.Duration{}}
}

func (f *fakeViolationBanCache) IncrementViolationCount(context.Context, int64, time.Duration) (int64, error) {
	return 0, nil
}
func (f *fakeViolationBanCache) ResetViolationCount(context.Context, int64) error { return nil }
func (f *fakeViolationBanCache) ClaimViolationNotifyCooldown(context.Context, int64, time.Duration) (bool, error) {
	return true, nil
}
func (f *fakeViolationBanCache) SetUserViolationBan(_ context.Context, userID int64, until time.Time, ttl time.Duration) error {
	f.bans[userID] = until
	f.sets[userID] = ttl
	return nil
}
func (f *fakeViolationBanCache) GetUserViolationBan(_ context.Context, userID int64) (time.Time, bool, error) {
	until, ok := f.bans[userID]
	return until, ok, f.getErr
}
func (f *fakeViolationBanCache) GetUserViolationBans(_ context.Context, userIDs []int64) (map[int64]time.Time, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	result := make(map[int64]time.Time, len(userIDs))
	for _, id := range userIDs {
		if until, ok := f.bans[id]; ok {
			result[id] = until
		}
	}
	return result, nil
}
func (f *fakeViolationBanCache) ClearUserViolationBan(_ context.Context, userID int64) error {
	delete(f.bans, userID)
	return nil
}
func (f *fakeViolationBanCache) ClaimViolationDedup(context.Context, int64, string, time.Duration) (bool, error) {
	return true, nil
}

type listUsersAdminStub struct {
	service.AdminService
	users []service.User
}

func (s *listUsersAdminStub) ListUsers(context.Context, int, int, service.UserListFilters, string, string) ([]service.User, int64, error) {
	return s.users, int64(len(s.users)), nil
}

func TestAdminUserListIncludesViolationBanStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	until := time.Now().Add(30 * time.Minute).Truncate(time.Second)
	svc := &listUsersAdminStub{users: []service.User{
		{ID: 1, Email: "banned@test.com", Status: service.StatusActive},
		{ID: 2, Email: "clean@test.com", Status: service.StatusActive},
	}}

	newRouter := func(cache service.ViolationCounterCache) *gin.Engine {
		r := gin.New()
		h := NewUserHandler(svc, nil, nil, nil, nil, nil, nil)
		h.SetViolationBanCache(cache)
		r.GET("/admin/users", h.List)
		return r
	}

	t.Run("banned user carries violation_ban_until", func(t *testing.T) {
		cache := newFakeViolationBanCache()
		cache.bans[1] = until
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		newRouter(cache).ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"violation_ban_until"`)
		require.Contains(t, w.Body.String(), `"banned@test.com"`)
	})

	t.Run("cache not injected omits field and list still works", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		newRouter(nil).ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.NotContains(t, w.Body.String(), `"violation_ban_until"`)
		require.Contains(t, w.Body.String(), `"clean@test.com"`)
	})

	t.Run("cache error fails open", func(t *testing.T) {
		cache := newFakeViolationBanCache()
		cache.getErr = errors.New("redis down")
		cache.bans[1] = until
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		newRouter(cache).ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.NotContains(t, w.Body.String(), `"violation_ban_until"`)
	})
}

func TestAdminUserCreateViolationBan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newRouter := func(cache service.ViolationCounterCache) *gin.Engine {
		r := gin.New()
		h := NewUserHandler(newStubAdminService(), nil, nil, nil, nil, nil, nil)
		h.SetViolationBanCache(cache)
		r.POST("/admin/users/:id/violation-ban", h.CreateViolationBan)
		return r
	}

	t.Run("valid duration sets ban via cache", func(t *testing.T) {
		cache := newFakeViolationBanCache()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/users/7/violation-ban", strings.NewReader(`{"duration_minutes": 30}`))
		req.Header.Set("Content-Type", "application/json")
		newRouter(cache).ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 30*time.Minute, cache.sets[7])
		require.Contains(t, w.Body.String(), `"banned":true`)
		require.Contains(t, w.Body.String(), `"until"`)
	})

	t.Run("duration out of range rejected", func(t *testing.T) {
		cache := newFakeViolationBanCache()
		for _, body := range []string{`{"duration_minutes": 0}`, `{"duration_minutes": 10081}`, `{"duration_minutes": -5}`} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/admin/users/7/violation-ban", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			newRouter(cache).ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code, body)
		}
		require.Empty(t, cache.sets)
	})

	t.Run("cache unavailable returns 503", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/users/7/violation-ban", strings.NewReader(`{"duration_minutes": 30}`))
		req.Header.Set("Content-Type", "application/json")
		newRouter(nil).ServeHTTP(w, req)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("invalid body rejected", func(t *testing.T) {
		cache := newFakeViolationBanCache()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/users/7/violation-ban", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		newRouter(cache).ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
