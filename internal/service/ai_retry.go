package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	aiRetryMaxAttempts = 3
	aiRetryBaseDelay   = 200 * time.Millisecond
)

// isRetryableOpenAIError reports whether err is worth retrying.
// Transient categories: 429 rate limit, 5xx server errors, network failures,
// and ctx-deadline exceeded propagated through the openai SDK.
func isRetryableOpenAIError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.HTTPStatusCode {
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		}
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Timeouts and other network errors are typically transient.
		return true
	}
	return false
}

// retryAPI runs fn with exponential backoff while errors are retryable and the
// context is alive. The first call has no delay; subsequent attempts wait
// aiRetryBaseDelay * 2^(attempt-1).
func retryAPI(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < aiRetryMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetryableOpenAIError(err) {
			return err
		}
		if attempt+1 == aiRetryMaxAttempts {
			break
		}
		delay := aiRetryBaseDelay << attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

// withDefaultAIDeadline returns ctx with a 30s timeout when the caller has not
// supplied one. Prevents an OpenAI hang from blocking the request handler
// indefinitely. The returned cancel must always be invoked.
func withDefaultAIDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 30*time.Second)
}
