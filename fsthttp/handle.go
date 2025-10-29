// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"context"
	"fmt"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// Serve calls h, providing it with a context that will be canceled when Serve
// returns, a Request representing the incoming client request that initiated
// this execution, and a ResponseWriter that can be used to respond to that
// request. Serve will ensure the ResponseWriter has been closed before
// returning, and so should only be called once per execution.
func Serve(h Handler) {
	abireq, abibody, err := fastly.BodyDownstreamGet()
	if err != nil {
		panic(fmt.Errorf("get client handles: %w", err))
	}

	serve(h, abireq, abibody)

	// wait for any stale-while-revalidate goroutines to complete.
	guestCacheSWRPending.Wait()
}

func serve(h Handler, abireq *fastly.HTTPRequest, abibody *fastly.HTTPBody) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientRequest, err := newClientRequest(abireq, abibody)
	if err != nil {
		panic(fmt.Errorf("create client Request: %w", err))
	}
	clientResponseWriter, err := newResponseWriter()
	if err != nil {
		panic(fmt.Errorf("create client ResponseWriter: %w", err))
	}

	h.ServeHTTP(ctx, clientResponseWriter, clientRequest)
	clientResponseWriter.Close()
}

// ServeManyOptions controls the exit conditions for ServeMany.  Note that Compute might terminate the request hander earlier.
type ServeManyOptions struct {
	// NextTimeout is the amount of time to wait for the next request
	NextTimeout time.Duration

	// MaxLifetime is the total max lifetime for an instance
	MaxLifetime time.Duration

	// MaxRequests is the maximum number of requests to serve for a single instance
	MaxRequests int

	// Continue is a function that determines whether to continue
	// serving requests after the other conditions have been checked.
	// If Continue returns false, ServeMany will exit.  If Continue is
	// nil or returns true, ServeMany will continue.
	Continue func() bool
}

// ServeMany allows a single Compute instance to handle multiple requests.
func ServeMany(h HandlerFunc, serveOpts *ServeManyOptions) {
	start := time.Now()

	abireq, abibody, err := fastly.BodyDownstreamGet()
	if err != nil {
		panic(fmt.Errorf("get client handles: %w", err))
	}
	serve(h, abireq, abibody)

	// Serve the rest
	var requests int
	for {
		requests++
		if serveOpts.MaxRequests != 0 && requests > serveOpts.MaxRequests {
			break
		}

		if serveOpts.MaxLifetime != 0 && time.Since(start) > serveOpts.MaxLifetime {
			break
		}

		if serveOpts.Continue != nil && !serveOpts.Continue() {
			break
		}

		var opts fastly.NextRequestOptions
		if serveOpts.NextTimeout != 0 {
			opts.Timeout(serveOpts.NextTimeout)
		}

		promise, err := fastly.DownstreamNextRequest(&opts)
		if err != nil {
			panic(fmt.Errorf("get next request promise: %w", err))
		}

		abireq, abibody, err := promise.Wait()
		if err != nil {
			if status, ok := fastly.IsFastlyError(err); ok && status == fastly.FastlyStatusNone {
				break
			}
			panic(fmt.Errorf("get client handles: %w", err))
		}

		serve(h, abireq, abibody)
	}

	// wait for any stale-while-revalidate goroutines to complete.
	guestCacheSWRPending.Wait()
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
