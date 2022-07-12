// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		// Reset the URI and Host header, so the request is
		// recognized and routed correctly at the origin.
		r.URL.Scheme, r.URL.Host = "https", "httpbin.org"
		r.Header.Set("host", "httpbin.org")

		// Make sure the response isn't cached.
		r.CacheOptions.Pass = true

		// This requires your service to be configured with a backend
		// named "httpbin" and pointing to "https://httpbin.org".
		resp, err := r.Send(ctx, "httpbin")
		if err != nil {
			w.WriteHeader(fsthttp.StatusBadGateway)
			fmt.Fprintln(w, err.Error())
			return
		}

		w.Header().Reset(resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
