//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"bufio"
	"context"
	"io"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestByteRepeater(t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/byte_repeater", nil)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			t.Errorf("NewRequest: %v", err)
			return
		}
		req.CacheOptions.Pass = true

		resp, err := req.Send(ctx, "TheOrigin")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			t.Errorf("Send: %v", err)
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
				t.Errorf("ReadByte: %v", err)
				return
			}
		}
	}

	r, err := fsthttp.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	w := fsttest.NewRecorder()

	handler(context.Background(), w, r)

	if got, want := w.Code, fsthttp.StatusOK; got != want {
		t.Errorf("Code = %v; want %v", got, want)
	}

	if got, want := w.Body.String(), "112233445566778899001122\n\n"; got != want {
		t.Errorf("Body = %q; want %q", got, want)
	}
}
