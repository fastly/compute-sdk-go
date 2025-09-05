// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	var requests int
	fsthttp.ServeMany(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		requests++
		fmt.Fprintf(w, "Request %v, Hello, %s (%q, %q)!\n", requests, r.RemoteAddr, os.Getenv("FASTLY_TRACE_ID"), r.RequestID)
	}, &fsthttp.ServeManyOptions{
		NextTimeout: 1 * time.Second,
		MaxRequests: 100,
		MaxLifetime: 5 * time.Second,
	})
}
