//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func TestAsyncSelect(t *testing.T) {
	type requestInfo struct {
		url     string
		backend string
		header  string
	}

	var mu sync.Mutex
	headers := make(map[string]string)

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

			resp, err := req.Send(context.Background(), ri.backend)
			if err != nil {
				t.Errorf("%s: send request: %v", ri.url, err)
				return
			}

			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			mu.Lock()
			headers[ri.header] = resp.Header.Get(ri.header)
			mu.Unlock()
		}(ri)
	}
	wg.Wait()

	if got, want := headers["fooname"], "FooValue"; got != want {
		t.Errorf("Header[fooname] = %q, want %q", got, want)
	}

	if got, want := headers["barname"], "BarValue"; got != want {
		t.Errorf("Header[barname] = %q, want %q", got, want)
	}
}
