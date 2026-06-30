// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fastly/compute-sdk-go/device"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		d, err := device.Lookup(r.Header.Get("User-Agent"))
		if err != nil {
			log.Println("error during device lookup:", err)
			fsthttp.Error(w, fsthttp.StatusText(fsthttp.StatusInternalServerError), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "%+v\n", d)
	})
}
