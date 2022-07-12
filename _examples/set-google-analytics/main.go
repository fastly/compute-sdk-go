// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		resp, err := r.Send(ctx, "backend")
		if err != nil {
			w.WriteHeader(fsthttp.StatusBadGateway)
			fmt.Fprintln(w, err.Error())
			return
		}
		c, err := r.Cookie("_ga")
		if r.Header.Get("Fastly-FF") == "" && (err != nil || !strings.HasPrefix(c.Value, "GA")) {
			now := time.Now()
			rand.Seed(now.UnixNano())

			// The _ga cookie is made up of four fields:
			//
			// 1. Version number (GA1).
			// 2. Number of components in the domain separated by dot.
			// 3. Random unique ID.
			// 4. Time stamp.

			host := r.Header.Get("Host")
			numSegs := strings.Count(host, ".") + 1
			i := rand.Intn(2147483647-1000000000) + 1000000000
			value := fmt.Sprintf("GA1.%d.%d.%d", numSegs, i, now.Unix())

			cookie := &fsthttp.Cookie{
				Name:   "_ga",
				Value:  value,
				Domain: "." + host,
				MaxAge: 3600 * 24 * 365 * 2, // two years in seconds
			}

			fsthttp.SetCookie(resp.Header, cookie)

			// Prevent browser from caching a set-cookie
			resp.Header.Set("Cache-Control", "no-store, private")
		}

		w.Header().Reset(resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
