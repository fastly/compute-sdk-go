// Copyright 2023 Fastly, Inc.

package main

import (
	"context"
	"fmt"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	// Increase URL length limit to 16K
	fsthttp.RequestLimits.SetMaxURLLen(16 * 1024)

	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		fmt.Fprintf(w, "The length of the URL is %d\n", len(r.URL.String()))
	})
}
