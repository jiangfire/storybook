package main

import (
	"context"
	"testing"
	"time"
)

// TestContextWithTimeoutOrSignal 验证超时和信号处理函数
func TestContextWithTimeoutOrSignal(t *testing.T) {
	t.Run("带超时的context", func(t *testing.T) {
		timeout := 100 * time.Millisecond
		ctx, cancel := contextWithTimeoutOrSignal(timeout)
		defer cancel()

		// 验证context有超时
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining > timeout || remaining < timeout-10*time.Millisecond {
				t.Errorf("期望deadline接近 %v, 实际 %v", timeout, remaining)
			}
		} else {
			t.Error("期望context有deadline")
		}
	})

	t.Run("不带超时的context", func(t *testing.T) {
		ctx, cancel := contextWithTimeoutOrSignal(0)
		defer cancel()

		// 不带超时的context不应该有deadline
		if _, ok := ctx.Deadline(); ok {
			t.Error("不带超时的context不应该有deadline")
		}

		// 但应该仍然是可取消的
		if ctx.Err() != nil {
			t.Errorf("新创建的context不应该有错误: %v", ctx.Err())
		}
	})

	t.Run("超时后取消", func(t *testing.T) {
		timeout := 10 * time.Millisecond
		ctx, cancel := contextWithTimeoutOrSignal(timeout)
		defer cancel()

		// 等待超时
		<-ctx.Done()

		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("期望 DeadlineExceeded, 实际 %v", ctx.Err())
		}
	})
}

