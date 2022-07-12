// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		// Log to the console (`fastly logs tail`) and the client.
		log := log.New(io.MultiWriter(os.Stdout, w), "", log.Ltime)
		log.Printf("Starting")
		begin := time.Now()

		// Send several requests in parallel.
		var wg sync.WaitGroup
		for _, url := range []string{
			"https://httpbin.org/drip?delay=4&duration=1", // delay 4s + stream response body 1s = 5s
			"https://httpbin.org/drip?delay=2&duration=2", // delay 2s + stream response body 2s = 4s
			"https://httpbin.org/delay/3",                 // delay 3s + stream response body 0s = 3s
		} {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				log.Printf("Starting %s", url)

				req, err := fsthttp.NewRequest(fsthttp.MethodGet, url, nil)
				if err != nil {
					log.Printf("%s: create request: %v", url, err)
					return
				}
				req.CacheOptions.Pass = true

				// Sending HTTP requests in separate goroutines is both
				// concurrent and parallel. For example, 3 requests that each
				// take 3s to return a response will take about 3s in total.
				resp, err := req.Send(ctx, "httpbin")
				if err != nil {
					log.Printf("%s: send request: %v", url, err)
					return
				}

				// All other code run in separate goroutines is concurrent but
				// not parallel. For example, reading 3 response bodies that
				// each take 3s will take about 9s in total.
				_, err = io.Copy(io.Discard, resp.Body)
				if err != nil {
					log.Printf("%s: stream response body: %v", url, err)
					return
				}

				log.Printf("Finished %s", url)
			}(url)
		}
		wg.Wait()

		// All requests should finish in about as long as the longest individual
		// request took. That is, about 5s, rather than 5s+4s+3s=12s.
		log.Printf("Finished after %s", time.Since(begin))
	})
}
