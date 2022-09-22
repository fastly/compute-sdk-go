// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		if r.Method != "POST" {
			fsthttp.Error(w, "Method: want POST, have "+r.Method, fsthttp.StatusMethodNotAllowed)
			return
		}

		url := "http://example.org/hello"
		if r.URL.String() != url {
			fsthttp.Error(w, "URL: want "+url+", have: "+r.URL.String()+"\n", fsthttp.StatusBadRequest)
			return
		}

		localhost := "127.0.0.1"
		if r.RemoteAddr != localhost {
			fsthttp.Error(w, "RemoteAddr: want "+localhost+", have "+r.RemoteAddr+"\n", fsthttp.StatusBadRequest)
			return
		}

		w.Header().Apply(r.Header.Clone())

		io.Copy(w, r.Body)
	})
}
