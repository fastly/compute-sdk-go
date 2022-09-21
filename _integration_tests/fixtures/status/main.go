// Copyright 2022 Fastly, Inc.

package main

import (
	"context"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		fsthttp.Error(w, "Unauthorized", fsthttp.StatusUnauthorized)
	})
}
