// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"log"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		begin := time.Now()

		// Create a context with a 1-second timeout.
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		// Create the request, and set pass to true, to avoid caching.
		req, err := fsthttp.NewRequest(fsthttp.MethodGet, "https://http-me.glitch.me/wait=3000", nil)
		if err != nil {
			log.Printf("create request: %v", err)
			return
		}
		req.CacheOptions.Pass = true

		// This request takes 3 seconds to complete but should error after 1
		// second. It also requires your service to be configured with a backend
		// named "httpme" and pointing to "https://http-me.glitch.me".
		_, err = req.Send(ctx, "httpme")
		if err != nil {
			log.Printf("send request errored after %s: %v", time.Since(begin), err)
			return
		}

		// This line should not print because the request should have errored
		// before it completed.
		log.Printf("Finished after %s", time.Since(begin))
	})
}
