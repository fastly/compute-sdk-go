//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func TestByteRepeater(t *testing.T) {
	req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/byte_repeater", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.CacheOptions.Pass = true

	resp, err := req.Send(context.Background(), "TheOrigin")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	defer resp.Body.Close()

	var got bytes.Buffer
	br := bufio.NewReader(resp.Body)
loop:
	for {
		b, err := br.ReadByte()
		switch {
		case err == nil: // normal case
			got.Write([]byte{b, b})
		case errors.Is(err, io.EOF): // done
			break loop
		default: // error
			t.Fatalf("ReadByte: %v", err)
		}
	}

	if want := "112233445566778899001122\n\n"; got.String() != want {
		t.Errorf("Body = %q; want %q", got.String(), want)
	}
}
