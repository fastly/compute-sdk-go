// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		if r.Method != "POST" {
			w.WriteHeader(fsthttp.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Method: want POST, have %s\n", r.Method)
			return
		}

		url := "http://example.org/hello"
		if r.URL.String() != url {
			w.WriteHeader(fsthttp.StatusBadRequest)
			fmt.Fprintf(w, "URL: want %s, have: %s\n", url, r.URL.String())
			return
		}

		localhost := "127.0.0.1"
		if r.RemoteAddr != localhost {
			w.WriteHeader(fsthttp.StatusBadRequest)
			fmt.Fprintf(w, "RemoteAddr: want %s, have %s\n", localhost, r.RemoteAddr)
			return
		}

		w.Header().Apply(r.Header.Clone())

		io.Copy(w, r.Body)
	})
}
