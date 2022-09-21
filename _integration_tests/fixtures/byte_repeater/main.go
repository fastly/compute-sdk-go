// Copyright 2022 Fastly, Inc.

package main

import (
	"bufio"
	"context"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/byte_repeater", nil)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}
		req.CacheOptions.Pass = true

		resp, err := req.Send(ctx, "TheOrigin")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		br := bufio.NewReader(resp.Body)
		for {
			b, err := br.ReadByte()
			switch {
			case err == nil: // normal case
				w.Write([]byte{b, b})
			case err == io.EOF: // done
				return
			case err != nil: // error
				fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
				return
			}
		}
	})
}
