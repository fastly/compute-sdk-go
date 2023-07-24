//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestRequestUpstream(t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, _ *fsthttp.Request) {
		// Create our upstream request
		req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/request_upstream", nil)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			t.Errorf("NewRequest: %v", err)
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
			t.Errorf("Send: %v", err)
			return
		}

		w.Header().Reset(resp.Header.Clone())
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}

	r, err := fsthttp.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := fsttest.NewRecorder()

	handler(context.Background(), w, r)

	if got, want := w.Code, fsthttp.StatusOK; got != want {
		t.Errorf("Code = %d, want %d", got, want)
	}

	if got, want := w.Header().Get("OriginHeader"), "OriginValue"; got != want {
		t.Errorf("Header[OriginHeader] = %q, want %q", got, want)
	}

	if got, want := w.Header().Get("x-cat"), "meow, nyan, mrrow, miau"; got != want {
		t.Errorf("Header[x-cat] = %q, want %q", got, want)
	}

	if got, want := w.Body.String(), "Hello from Origin"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
