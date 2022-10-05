// Copyright 2022 Fastly, Inc.

package fstctx

import (
	"context"
	"time"
)

// WithDeadline returns a copy of the parent context which will be cancelled
// when the deadline arrives. Unlike context.WithDeadline the deadline will not
// be adjusted, and it will not return the context.DeadlineExceeded error.
// Canceling before the deadline will not clean up the spawned goroutine however
// the goroutine will become a no-op. It is possible for pauses in Wasm
// execution to cause the cancellation to happen after the deadline. This should
// not be relied on for applications where a precise deadline is required.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func WithDeadline(parent context.Context, d time.Time) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, time.Until(d))
}

// WithTimeout returns a copy of the parent context which will be cancelled
// after the provided timeout. Unlike context.WithTimeout the deadline will not
// be adjusted, and it will not return the context.DeadlineExceeded error.
// Canceling before the timeout will not clean up the spawned goroutine
// however the goroutine will become a no-op. It is possible for pauses in Wasm
// execution to cause the timeout to be delayed. This should not be relied on
// for applications where a precise timout is required.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete:
//
//	func slowOperationWithTimeout(ctx context.Context) (Result, error) {
//		ctx, cancel := fstctx.WithTimeout(ctx, 100*time.Millisecond)
//		defer cancel()  // releases resources if slowOperation completes before timeout elapses
//		return slowOperation(ctx)
//	}
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		time.Sleep(timeout)
		if ctx.Err() == nil {
			cancel()
		}
	}()

	return ctx, cancel
}
