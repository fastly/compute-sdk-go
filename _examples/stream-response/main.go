// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		n := getQueryInt(r, "n", 5)
		d := getQueryDuration(r, "d", 250*time.Millisecond)

		// If you're using cURL, be sure to use `-N, --no-buffer`.
		fmt.Fprintf(w, "n=%d, d=%s\n", n, d)
		for i := 1; i <= n; i++ {
			time.Sleep(d)
			fmt.Fprintf(w, " ʕ◔ϖ◔ʔ")
		}
		fmt.Fprintln(w)
	})
}

func getQueryInt(r *fsthttp.Request, key string, def int) int {
	i, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return i
}

func getQueryDuration(r *fsthttp.Request, key string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return d
}
