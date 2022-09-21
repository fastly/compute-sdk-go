// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"

	"github.com/fastly/compute-sdk-go/edgedict"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, _ *fsthttp.Request) {
		d, err := edgedict.Open("edge_dictionary")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		twitter, err := d.Get("twitter")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, twitter)
	})
}
