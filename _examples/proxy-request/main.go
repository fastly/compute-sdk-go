// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		// Reset the URI and Host header, so the request is
		// recognized and routed correctly at the origin.
		r.URL.Scheme, r.URL.Host = "https", "http-me.glitch.me"
		r.Header.Set("host", "http-me.glitch.me")

		// Make sure the response isn't cached.
		r.CacheOptions.Pass = true

		// This requires your service to be configured with a backend
		// named "httpme" and pointing to "https://http-me.glitch.me".
		resp, err := r.Send(ctx, "httpme")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		w.Header().Reset(resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
