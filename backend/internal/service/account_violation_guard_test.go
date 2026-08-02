package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeAccountViolationGuard struct {
	decision *AccountGuardDecision
	err      error
	calls    int
	input    AccountGuardCheckInput
}

func (f *fakeAccountViolationGuard) Check(_ context.Context, input AccountGuardCheckInput) (*AccountGuardDecision, error) {
	f.calls++
	f.input = input
	return f.decision, f.err
}

func newGuardTestGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return c, recorder
}

func TestRunAccountViolationGuard_NilGuardAllows(t *testing.T) {
	c, _ := newGuardTestGinContext()
	decision := runAccountViolationGuard(nil, context.Background(), c, &Account{ID: 1}, "anthropic_messages", "m", []byte(`{}`))
	require.Nil(t, decision)
}

func TestRunAccountViolationGuard_CheckErrorFailsOpen(t *testing.T) {
	guard := &fakeAccountViolationGuard{err: errors.New("config store unavailable")}
	c, _ := newGuardTestGinContext()
	decision := runAccountViolationGuard(guard, context.Background(), c, &Account{ID: 1}, "anthropic_messages", "m", []byte(`{}`))
	require.Nil(t, decision)
	require.Equal(t, 1, guard.calls)
}

func TestGatewayServiceCheckAccountViolationGuard_BlocksWith403NoFailover(t *testing.T) {
	guard := &fakeAccountViolationGuard{decision: &AccountGuardDecision{Blocked: true, Reason: "violent"}}
	svc := &GatewayService{accountGuard: guard}
	c, recorder := newGuardTestGinContext()
	account := &Account{ID: 7, Name: "acc", Platform: PlatformAnthropic}

	err := svc.checkAccountViolationGuard(context.Background(), c, account, "anthropic_messages", "claude-test", []byte(`{"messages":[]}`))
	require.Error(t, err)

	var blockedErr *AccountGuardBlockedError
	require.True(t, errors.As(err, &blockedErr))
	require.Equal(t, int64(7), blockedErr.AccountID)

	// 关键不变量：该错误绝不能被识别为可换号的 failover 错误
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr))

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "content_policy_violation")
	require.Contains(t, recorder.Body.String(), `"type":"error"`) // Anthropic 错误信封
}

func TestOpenAIGatewayServiceCheckAccountViolationGuard_BlocksWith403NoFailover(t *testing.T) {
	guard := &fakeAccountViolationGuard{decision: &AccountGuardDecision{Blocked: true, Reason: "jailbreak"}}
	svc := &OpenAIGatewayService{accountGuard: guard}
	c, recorder := newGuardTestGinContext()
	account := &Account{ID: 9, Name: "oai-acc", Platform: PlatformOpenAI}

	err := svc.checkAccountViolationGuard(context.Background(), c, account, "", "gpt-test", []byte(`{"model":"gpt-test"}`))
	require.Error(t, err)

	var blockedErr *AccountGuardBlockedError
	require.True(t, errors.As(err, &blockedErr))

	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr))

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "content_policy_violation")
	require.NotContains(t, recorder.Body.String(), `"type":"error"`) // OpenAI 错误信封（无外层 type:error）
}

func TestGatewayServiceCheckAccountViolationGuard_AllowPassesThrough(t *testing.T) {
	guard := &fakeAccountViolationGuard{decision: &AccountGuardDecision{Blocked: false}}
	svc := &GatewayService{accountGuard: guard}
	c, recorder := newGuardTestGinContext()

	err := svc.checkAccountViolationGuard(context.Background(), c, &Account{ID: 7}, "anthropic_messages", "m", []byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, recorder.Code)
}
