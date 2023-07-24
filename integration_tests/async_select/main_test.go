//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestAsyncSelect(t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		type requestInfo struct {
			url     string
			backend string
			header  string
		}

		// Send several requests in parallel.
		var wg sync.WaitGroup
		for _, ri := range []requestInfo{
			{"https://compute-sdk-test-backend.edgecompute.app/async_select_1", "TheOrigin", "fooname"},
			{"https://compute-sdk-test-backend.edgecompute.app/async_select_2", "TheOrigin2", "barname"},
		} {
			wg.Add(1)
			go func(ri requestInfo) {
				defer wg.Done()

				req, err := fsthttp.NewRequest("GET", ri.url, nil)
				if err != nil {
					t.Errorf("%s: create request: %v", ri.url, err)
					return
				}
				req.CacheOptions.Pass = true

				resp, err := req.Send(ctx, ri.backend)
				if err != nil {
					t.Errorf("%s: send request: %v", ri.url, err)
					return
				}

				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				w.Header().Set(ri.header, resp.Header.Get(ri.header))
			}(ri)
		}
		wg.Wait()

		fmt.Fprintf(w, "pong")
	}

	r, err := fsthttp.NewRequest("POST", "/hello", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	w := fsttest.NewRecorder()

	handler(context.Background(), w, r)

	if got, want := w.Code, fsthttp.StatusOK; got != want {
		t.Errorf("Code = %d, want %d", got, want)
	}

	if got, want := w.Header().Get("FooName"), "FooValue"; got != want {
		t.Errorf("Header[FooName] = %q, want %q", got, want)
	}

	if got, want := w.Header().Get("BarName"), "BarValue"; got != want {
		t.Errorf("Header[BarName] = %q, want %q", got, want)
	}

	if got, want := w.Body.String(), "pong"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
