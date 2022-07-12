// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/rtlog"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		fmt.Fprintln(os.Stdout, "os.Stdout can be streamed via `fastly logs tail`")
		fmt.Fprintln(os.Stderr, "os.Stderr can be streamed via `fastly logs tail`")

		endpoint := rtlog.Open("my-logging-endpoint")
		fmt.Fprintln(endpoint, "Real-time logging is available via `package rtlog`")

		mw := io.MultiWriter(os.Stdout, endpoint, w)
		fmt.Fprintln(mw, "Mix-and-match destinations with helpers like io.MultiWriter")

		fmt.Fprintln(mw, "Several environment variables are defined by default...")
		for _, key := range []string{
			"FASTLY_CACHE_GENERATION",
			"FASTLY_CUSTOMER_ID",
			"FASTLY_HOSTNAME",
			"FASTLY_POP",
			"FASTLY_REGION",
			"FASTLY_SERVICE_ID",
			"FASTLY_SERVICE_VERSION",
			"FASTLY_TRACE_ID",
		} {
			fmt.Fprintf(mw, "%s=%s\n", key, os.Getenv(key))
		}

		prefix := fmt.Sprintf("%s | %s | ", os.Getenv("FASTLY_SERVICE_VERSION"), r.RemoteAddr)
		logger := log.New(os.Stdout, prefix, log.LstdFlags|log.LUTC)
		logger.Printf("It can be useful to create a logger with request-specific metadata built in")
	})
}
