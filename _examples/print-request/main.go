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
		fmt.Fprintf(w, "Method:     %q\n", r.Method)
		fmt.Fprintf(w, "URL:        %v\n", r.URL)
		fmt.Fprintf(w, "Proto:      %q\n", r.Proto)
		fmt.Fprintf(w, "ProtoMajor: %d\n", r.ProtoMajor)
		fmt.Fprintf(w, "ProtoMinor: %d\n", r.ProtoMinor)
		fmt.Fprintf(w, "RemoteAddr: %q\n", r.RemoteAddr)
		fmt.Fprintf(w, "TLSInfo:    %#v\n", r.TLSInfo)

		fmt.Fprintf(w, "\n")

		for _, k := range r.Header.Keys() {
			fmt.Fprintf(w, "%s: %v\n", k, r.Header.Get(k))
		}

		fmt.Fprintf(w, "\n")

		io.Copy(w, r.Body)
	})
}
