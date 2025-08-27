// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	var requests int
	fsthttp.ServeMany(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		requests++
		fmt.Fprintf(w, "Request %v, Hello, %s!\n", requests, r.RemoteAddr)
	}, &fsthttp.ServeManyOptions{
		NextTimeout: 10 * time.Second,
		MaxRequests: 100,
		MaxLifetime: 10 * time.Second,
	})
}
