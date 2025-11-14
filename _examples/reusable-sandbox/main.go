// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeMany(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		meta, err := r.FastlyMeta()
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Request %v, Hello, %s (sandbox: %q, request: %q)!\n", meta.SandboxRequests, r.RemoteAddr, meta.SandboxID, meta.RequestID)
	}, &fsthttp.ServeManyOptions{
		NextTimeout: 1 * time.Second,
		MaxRequests: 100,
		MaxLifetime: 5 * time.Second,
	})
}
