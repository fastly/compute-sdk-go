//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestDownstreamRequest(t *testing.T) {
	// This uses fsthttp.ServeFunc() to test an incoming request.
	// Viceroy constructs a simple GET http://example.com request with
	// the remote address being 127.0.0.1, so that's what we check for
	// here.
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %s, want GET", r.Method)
			return
		}

		url := "http://example.com/"
		if r.URL.String() != url {
			t.Errorf("URL = %s, want %s", r.URL.String(), url)
			return
		}

		localhost := "127.0.0.1"
		if r.RemoteAddr != localhost {
			t.Errorf("RemoteAddr = %s, want %s", r.RemoteAddr, localhost)
			return
		}
	})
}

func TestDownstreamResponse(t *testing.T) {
	// In this test we construct our own request and response recorder
	// to test that the headers and body on the response are sent
	// properly.
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		if r.Method != "POST" {
			fsthttp.Error(w, fmt.Sprintf("Method = %s, want POST", r.Method), fsthttp.StatusMethodNotAllowed)
			return
		}

		w.Header().Apply(r.Header.Clone())
		io.Copy(w, r.Body)
	}

	const body = "downstream requeest!"
	r, err := fsthttp.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	r.Header.Set("DownstreamName", "DownstreamValue")

	w := fsttest.NewRecorder()

	handler(context.Background(), w, r)

	if got, want := w.Code, fsthttp.StatusOK; got != want {
		t.Errorf("Code = %d, want %d", got, want)
	}

	if got, want := w.Header().Get("DownstreamName"), "DownstreamValue"; got != want {
		t.Errorf("Header[DownstreamName] = %q, want %q", got, want)
	}

	if got, want := w.Body.String(), body; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
