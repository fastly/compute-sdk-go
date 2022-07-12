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

		// Determine the framing headers (Content-Length/Transfer-Encoding)
		// based on the message body (default)
		r.ManualFramingMode = false

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

		// Use the framing headers set in the message.
		w.SetManualFramingMode(true)

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
