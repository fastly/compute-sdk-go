//go:build !test

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		// Verify that a bunch of function calls work, then return OK.
		fmt.Println("Proto =", r.Proto)
		fmt.Println("-- Headers --")
		for k, v := range r.Header {
			fmt.Printf("%s: %s\n", k, v)
		}
		fmt.Println("--")
		var body bytes.Buffer
		n, err := io.Copy(&body, r.Body)
		if err != nil {
			panic(err)
		}
		if n > 0 {
			fmt.Printf("Body =\n%s\n", body.String())
		}
		fm, err := r.FastlyMeta()
		if err != nil {
			panic(err)
		} else if fm == nil {
			panic("FastlyMeta() returned nil")
		}
		fmt.Printf("FastlyMeta() = %+v\n", fm)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Test-Header", "present")
		w.Write([]byte("OK"))
	}
	fsthttp.ServeFunc(handler)
}
