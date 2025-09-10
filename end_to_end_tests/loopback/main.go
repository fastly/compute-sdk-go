//go:build !test

package main

import (
	"context"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Test-Header", "present")
		w.Write([]byte("OK"))
	}
	fsthttp.ServeFunc(handler)
}
