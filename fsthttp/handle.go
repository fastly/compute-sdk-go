// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	serveOnce            sync.Once
	clientRequest        *Request
	clientResponseWriter ResponseWriter
)

// Serve calls h, providing it with a context that will be canceled when Serve
// returns, a Request representing the incoming client request that initiated
// this execution, and a ResponseWriter that can be used to respond to that
// request. Serve will ensure the ResponseWriter has been closed before
// returning, and so should only be called once per execution.
func Serve(h Handler) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	headerTooLong := false

	serveOnce.Do(func() {
		var err error
		clientRequest, err = newClientRequest()
		if err != nil {
			status, ok := fastly.IsFastlyError(errors.Unwrap(err))
			if ok && status == fastly.FastlyStatusBufLen {
				headerTooLong = true
			}
			if !headerTooLong {
				panic(fmt.Errorf("create client Request: %w", err))
			}
		}
		clientResponseWriter, err = newResponseWriter()
		if err != nil {
			panic(fmt.Errorf("create client ResponseWriter: %w", err))
		}
	})

	defer clientResponseWriter.Close()
	if headerTooLong {
		clientResponseWriter.WriteHeader(StatusRequestHeaderFieldsTooLarge)
		clientResponseWriter.Write([]byte("Request Header Fields Too Large\n"))
		return
	}
	h.ServeHTTP(ctx, clientResponseWriter, clientRequest)
}

// ServeFunc is sugar for Serve(HandlerFunc(f)).
func ServeFunc(f HandlerFunc) {
	Serve(f)
}

// Handler describes anything which can handle, or respond to, an HTTP request.
// It has the same semantics as net/http.Handler, but operates on the Request
// and ResponseWriter types defined in this package.
type Handler interface {
	ServeHTTP(ctx context.Context, w ResponseWriter, r *Request)
}

// HandlerFunc adapts a function to a Handler.
type HandlerFunc func(ctx context.Context, w ResponseWriter, r *Request)

// ServeHTTP implements Handler by calling f(ctx, w, r).
func (f HandlerFunc) ServeHTTP(ctx context.Context, w ResponseWriter, r *Request) {
	f(ctx, w, r)
}
