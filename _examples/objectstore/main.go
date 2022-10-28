// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/objectstore"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		o, err := objectstore.Open("example_objectstore")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		v, err := o.Lookup("foo")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		w.WriteHeader(fsthttp.StatusOK)
		io.Copy(w, v)
	})
}
