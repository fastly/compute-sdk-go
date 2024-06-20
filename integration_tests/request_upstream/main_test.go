//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

func TestRequestUpstream(t *testing.T) {
	t.Run("useAppend=false", func(t *testing.T) { requestUpstream(false, t) })
	t.Run("useAppend=true", func(t *testing.T) { requestUpstream(true, t) })
}

func requestUpstream(useAppend bool, t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, _ *fsthttp.Request) {
		// Create our upstream request
		req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/request_upstream", nil)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			t.Errorf("NewRequest: %v", err)
			return
		}

		req.Header.Set("UpstreamHeader", "UpstreamValue")

		// Make sure the response isn't cached.
		req.CacheOptions.Pass = true

		// This requires your service to be configured with a backend
		// named "TheOrigin" and pointing to "http://provider.org/TheURL".
		resp, err := req.Send(ctx, "TheOrigin")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			t.Errorf("Send: %v", err)
			return
		}

		w.Header().Reset(resp.Header.Clone())
		w.WriteHeader(resp.StatusCode)
		if useAppend {
			w.Append(resp.Body)
		} else {
			io.Copy(w, resp.Body)
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

	if got, want := w.Header().Get("OriginHeader"), "OriginValue"; got != want {
		t.Errorf("Header[OriginHeader] = %q, want %q", got, want)
	}

	if got, want := w.Header().Get("x-cat"), "meow, nyan, mrrow, miau"; got != want {
		t.Errorf("Header[x-cat] = %q, want %q", got, want)
	}

	if got, want := w.Body.String(), "Hello from Origin"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}

const bodySize = 64 * 1024

func TestRequestUpstreamBody(t *testing.T) {
	body := make([]byte, bodySize)
	for i := range body {
		body[i] = byte(i)
	}

	b, err := fastly.NewHTTPBody()
	if err != nil {
		t.Fatalf("NewHTTPBody: %v", err)
	}
	_, err = b.Write(body)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	testcases := []struct {
		name    string
		body    io.Reader
		size    int
		chunked bool
	}{
		{name: "nil", body: nil},
		{name: "bytes.Reader", body: bytes.NewReader(body), size: bodySize},
		{name: "bytes.Buffer", body: bytes.NewBuffer(body), size: bodySize},
		{name: "strings.Reader", body: strings.NewReader(string(body)), size: bodySize},
		{name: "io.NopCloser", body: io.NopCloser(bytes.NewReader(body)), chunked: true},
		{name: "fastly.HTTPBody", body: b, chunked: true},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			requestUpstreamBody(t, tc.body, tc.size, tc.chunked)
		})
	}
}

func requestUpstreamBody(t *testing.T, body io.Reader, size int, chunked bool) {
	req, err := fsthttp.NewRequest("POST", "https://http-me.glitch.me/?anything", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.CacheOptions.Pass = true

	resp, err := req.Send(context.Background(), "httpme")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	defer resp.Body.Close()

	var respData struct {
		Headers map[string]string `json:"headers"`
	}

	gotBody := new(bytes.Buffer)
	if err := json.NewDecoder(io.TeeReader(resp.Body, gotBody)).Decode(&respData); err != nil {
		t.Fatalf("Decode: %v\nBody:\n%s", err, gotBody.String())
	}

	var teWant, clWant string
	if chunked {
		teWant = "chunked"
	} else {
		clWant = strconv.Itoa(size)
	}

	if got, want := respData.Headers["transfer-encoding"], teWant; got != want {
		t.Errorf("Header[transfer-encoding] = %q, want %q", got, want)
	}
	if got, want := respData.Headers["content-length"], clWant; got != want {
		t.Errorf("Header[content-length] = %q, want %q", got, want)
	}
}
