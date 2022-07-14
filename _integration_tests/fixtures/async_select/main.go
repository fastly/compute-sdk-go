// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
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
					log.Printf("%s: create request: %v", ri.url, err)
					return
				}
				req.CacheOptions.Pass = true

				resp, err := req.Send(ctx, ri.backend)
				if err != nil {
					log.Printf("%s: send request: %v", ri.url, err)
					return
				}

				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				w.Header().Set(ri.header, resp.Header.Get(ri.header))
			}(ri)
		}
		wg.Wait()

		fmt.Fprintf(w, "pong")
	})
}
