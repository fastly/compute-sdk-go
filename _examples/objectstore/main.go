// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"
	"strings"

	"github.com/fastly/compute-sdk-go/fsthttp"
	kvstore "github.com/fastly/compute-sdk-go/kvstore"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		o, err := kvstore.Open("example_kvstore")
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

		// We can detect when a key does not exist and supply a default value instead.
		var reader io.Reader
		v, err = o.Lookup("might-not-exist")
		if err != nil && err == kvstore.ErrKeyNotFound {
			reader = strings.NewReader("default value")
		} else if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		} else {
			reader = v
		}

		io.Copy(w, reader)
	})
}
