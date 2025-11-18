//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func TestLoopback(t *testing.T) {
	t.Run("GET", func(t *testing.T) {
		req, err := fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp := doLoopbackRequest(t, req)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if b, want := string(body), "OK"; b != want {
			t.Errorf("resp.Body = %s, want: %s", b, want)
		}
	})

	t.Run("POST", func(t *testing.T) {
		req, err := fsthttp.NewRequest("POST", "http://anyplace.horse", strings.NewReader("hello there"))
		if err != nil {
			t.Fatal(err)
		}
		resp := doLoopbackRequest(t, req)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if b, want := string(body), "OK"; b != want {
			t.Errorf("resp.Body = %s, want: %s", b, want)
		}
	})
}

func doLoopbackRequest(t *testing.T, req *fsthttp.Request) *fsthttp.Response {
	t.Helper()
	resp, err := req.Send(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if h, want := resp.Header.Get("content-type"), "text/plain"; h != want {
		t.Errorf("Content-Type = %s, want: %s", h, want)
	}
	if resp.Header.Get("date") == "" {
		t.Errorf("expected default Date header is missing")
	}
	if h, want := resp.Header.Get("x-test-header"), "present"; h != want {
		t.Errorf("X-Test-Header = %s, want: %s", h, want)
	}
	return resp
}

func Test1xxStatusCode(t *testing.T) {
	req, err := fsthttp.NewRequest("GET", "http://anyplace.horse?status_code=101", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := req.Send(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, 500; got != want {
		// Unlike Compute, Viceroy returns a 101 status code until
		// https://github.com/fastly/Viceroy/pull/557 lands, probably in
		// v0.16.1.
		if got == 101 {
			t.Logf("StatusCode = %d, want: %d; accepting until Viceroy is fixed", got, want)
		} else {
			t.Errorf("StatusCode = %d, want: %d", got, want)
		}
	}
}
