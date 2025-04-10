//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"strconv"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

const backend = "httpme"

func TestInternalDebug(t *testing.T) {

	uri := "https://http-me.glitch.me/anything/" + strconv.Itoa(rand.Int()) + "/"
	req, err := fsthttp.NewRequest("GET", uri, nil)
	if err != nil {
		t.Errorf("error during NewRequest: uri=%v err=%v", uri, err)
		return
	}

	ctx := context.Background()

	req.Header.Add("foobar", "quxzot")

	req.ConstructABIRequest()

	resp, err := req.Send(ctx, backend)
	if err != nil {
		t.Errorf("error during Send: %v", err)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("readall(body)=%v, want %v", err, nil)
		return
	}

	if !bytes.Contains(body, []byte(`"foobar": "quxzot",`)) {
		t.Errorf("body missing foobar/quzot header: got %v", string(body))
		return
	}
}
