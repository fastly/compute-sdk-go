//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func isVowel(b byte) bool {
	switch b {
	case 'a', 'A', 'e', 'E', 'i', 'I', 'o', 'O', 'u', 'U':
		return true
	}
	return false
}

func TestStreamingClose(t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/streaming_close", nil)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		req.CacheOptions.Pass = true
		resp, err := req.Send(ctx, "TheOrigin")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		var removed int

		br := bufio.NewReader(resp.Body)
		for {
			b, err := br.ReadByte()
			if err == io.EOF {
				break
			}

			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
				t.Errorf("ReadByte: %v", err)
				return
			}

			if isVowel(b) {
				removed++
			} else {
				w.Write([]byte{b})
			}
		}

		w.Close()

		req2, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app", nil)
		if err != nil {
			t.Errorf("NewRequest: %v", err)
			return
		}

		req2.Header.Set("Vowels-Removed", strconv.Itoa(removed))
		if _, err = req2.Send(ctx, "TheOrigin2"); err != nil {
			t.Errorf("Send: %v", err)
			return
		}
	}

	r, err := fsthttp.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := fsttest.NewRecorder()

	handler(context.Background(), w, r)

	if got, want := w.Code, fsthttp.StatusOK; got != want {
		t.Errorf("Code = %d, want %d", got, want)
	}

	if got, want := w.Body.String(), "wll smth\n"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
