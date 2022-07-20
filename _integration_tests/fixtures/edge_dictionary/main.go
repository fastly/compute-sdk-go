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
			w.WriteHeader(fsthttp.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}

		twitter, err := d.Get("twitter")
		if err != nil {
			w.WriteHeader(fsthttp.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}

		fmt.Fprint(w, twitter)
	})
}
