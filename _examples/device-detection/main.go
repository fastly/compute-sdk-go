// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"

	"github.com/fastly/compute-sdk-go/device"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		d, err := device.Lookup(r.Header.Get("User-Agent"))
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "%+v\n", d)
	})
}
