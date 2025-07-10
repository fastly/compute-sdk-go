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
		r.URL.Scheme, r.URL.Host = "https", "http-me.fastly.dev"
		r.Header.Set("host", "http-me.fastly.dev")

		// Determine the framing headers (Content-Length/Transfer-Encoding)
		// based on the message body (default)
		r.ManualFramingMode = false

		// Make sure the response isn't cached.
		r.CacheOptions.Pass = true

		// This requires your service to be configured with a backend
		// named "httpme" and pointing to "https://http-me.fastly.dev".
		resp, err := r.Send(ctx, "httpme")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		w.Header().Reset(resp.Header)

		// Use the framing headers set in the message.
		w.SetManualFramingMode(true)

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
