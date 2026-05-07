package service

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func TestRetryAPI_FirstTrySuccess(t *testing.T) {
	calls := 0
	err := retryAPI(context.Background(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetryAPI_NonRetryableReturnsImmediately(t *testing.T) {
	calls := 0
	want := errors.New("permanent failure")
	err := retryAPI(context.Background(), func() error {
		calls++
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if calls != 1 {
		t.Errorf("expected non-retryable to stop after 1 call, got %d", calls)
	}
}

func TestRetryAPI_RetryableEventuallySucceeds(t *testing.T) {
	var calls int32
	rateLimit := &openai.APIError{HTTPStatusCode: 429, Message: "rate limited"}
	err := retryAPI(context.Background(), func() error {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			return rateLimit
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected eventual success, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetryAPI_GivesUpAfterMaxAttempts(t *testing.T) {
	var calls int32
	rateLimit := &openai.APIError{HTTPStatusCode: 503, Message: "unavailable"}
	err := retryAPI(context.Background(), func() error {
		atomic.AddInt32(&calls, 1)
		return rateLimit
	})
	if err == nil {
		t.Fatalf("expected error after max attempts")
	}
	if calls != aiRetryMaxAttempts {
		t.Errorf("expected %d calls, got %d", aiRetryMaxAttempts, calls)
	}
}

func TestRetryAPI_RespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := retryAPI(ctx, func() error {
		calls++
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 0 {
		t.Errorf("expected zero calls when ctx already cancelled, got %d", calls)
	}
}

type fakeNetErr struct{}

func (fakeNetErr) Error() string   { return "fake net err" }
func (fakeNetErr) Timeout() bool   { return true }
func (fakeNetErr) Temporary() bool { return true }

var _ net.Error = fakeNetErr{}

func TestIsRetryableOpenAIError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"rate-limit", &openai.APIError{HTTPStatusCode: 429}, true},
		{"server-500", &openai.APIError{HTTPStatusCode: 500}, true},
		{"bad-gateway", &openai.APIError{HTTPStatusCode: 502}, true},
		{"unavailable", &openai.APIError{HTTPStatusCode: 503}, true},
		{"gateway-timeout", &openai.APIError{HTTPStatusCode: 504}, true},
		{"client-400", &openai.APIError{HTTPStatusCode: 400}, false},
		{"unauthorized", &openai.APIError{HTTPStatusCode: 401}, false},
		{"forbidden", &openai.APIError{HTTPStatusCode: 403}, false},
		{"not-found", &openai.APIError{HTTPStatusCode: 404}, false},
		{"net-error", fakeNetErr{}, true},
		{"plain-error", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryableOpenAIError(tc.err); got != tc.want {
				t.Errorf("isRetryableOpenAIError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestWithDefaultAIDeadline_Applies(t *testing.T) {
	ctx, cancel := withDefaultAIDeadline(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected default deadline")
	}
	if time.Until(deadline) > 31*time.Second || time.Until(deadline) < 25*time.Second {
		t.Errorf("expected ~30s default deadline, got %s", time.Until(deadline))
	}
}

func TestWithDefaultAIDeadline_PreservesExisting(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx, cancel2 := withDefaultAIDeadline(parent)
	defer cancel2()

	deadline, _ := ctx.Deadline()
	if time.Until(deadline) > 6*time.Second {
		t.Errorf("expected to inherit caller's 5s deadline, got %s", time.Until(deadline))
	}
}

func TestAICache_StoreAndHit(t *testing.T) {
	t.Cleanup(InvalidateAIServiceCache)
	InvalidateAIServiceCache()

	updated := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	stored := storeAIServiceCache(&heuristicAIService{}, 42, updated)
	if stored == nil {
		t.Fatal("storeAIServiceCache returned nil")
	}

	hit := cachedAIServiceFor(42, updated)
	if hit != stored {
		t.Errorf("expected cache hit to return same instance")
	}
}

func TestAICache_DifferentConfigMisses(t *testing.T) {
	t.Cleanup(InvalidateAIServiceCache)
	InvalidateAIServiceCache()

	updated := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	storeAIServiceCache(&heuristicAIService{}, 1, updated)

	if cachedAIServiceFor(2, updated) != nil {
		t.Error("expected cache miss for different config id")
	}
	if cachedAIServiceFor(1, updated.Add(time.Second)) != nil {
		t.Error("expected cache miss for newer updated_at")
	}
}

func TestInvalidateAIServiceCache(t *testing.T) {
	t.Cleanup(InvalidateAIServiceCache)

	updated := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	storeAIServiceCache(&heuristicAIService{}, 1, updated)

	InvalidateAIServiceCache()

	if cachedAIServiceFor(1, updated) != nil {
		t.Error("expected cache to be empty after invalidate")
	}
}
