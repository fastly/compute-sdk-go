//go:build !test

package main

import (
	"context"
	"fmt"
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
		if r.Header.Get("Fresh-Sandbox") == "1" {
			opts.Continue = func() bool {
				return false
			}
		}

		meta, err := r.FastlyMeta()
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}
		sandboxID, requestID := meta.SandboxID, meta.RequestID
		fmt.Printf("Sandbox ID: %s, Request ID: %s\n", sandboxID, requestID)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Sandbox-ID", sandboxID)
		w.Header().Set("Request-ID", requestID)
		w.Write([]byte("OK"))
	}

	fsthttp.ServeMany(handler, opts)
}
