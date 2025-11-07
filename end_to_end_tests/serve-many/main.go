//go:build !test

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	opts := &fsthttp.ServeManyOptions{
		NextTimeout: 5 * time.Second,
		MaxRequests: 100,
		MaxLifetime: 10 * time.Second,
	}

	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		if r.Header.Get("Close-Session") == "1" {
			opts.Continue = func() bool {
				return false
			}
		}

		sessionID, requestID := os.Getenv("FASTLY_TRACE_ID"), r.RequestID
		fmt.Printf("Session ID: %s, Request ID: %s\n", sessionID, requestID)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Session-ID", sessionID)
		w.Header().Set("Request-ID", requestID)
		w.Write([]byte("OK"))
	}

	fsthttp.ServeMany(handler, opts)
}
