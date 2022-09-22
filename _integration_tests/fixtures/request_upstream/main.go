// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, _ *fsthttp.Request) {
		// Create our upstream request
		req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/request_upstream", nil)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		req.Header.Set("UpstreamHeader", "UpstreamValue")

		// Make sure the response isn't cached.
		req.CacheOptions.Pass = true

		// This requires your service to be configured with a backend
		// named "TheOrigin" and pointing to "http://provider.org/TheURL".
		resp, err := req.Send(ctx, "TheOrigin")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		w.Header().Reset(resp.Header.Clone())
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
