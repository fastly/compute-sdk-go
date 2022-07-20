// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/rtlog"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		fmt.Fprintln(os.Stdout, "os.Stdout can be streamed via `fastly logs tail`")
		fmt.Fprintln(os.Stderr, "os.Stderr can be streamed via `fastly logs tail`")

		endpoint := rtlog.Open("ComputeLog")
		fmt.Fprintln(endpoint, "Hello!")

		fmt.Fprintln(w, "Logging is done!")
	})
}
