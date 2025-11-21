//go:build wasip1 && !nofastlyhostcalls

package main

import (
	"context"
	"io"
	"os"
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

// Validate that accessors that (mostly) don't make sense against
// backend requests return sane values.
func TestBackendRequest(t *testing.T) {
	t.Run("FastlyMeta", func(t *testing.T) {
		req, err := fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
		if err != nil {
			t.Fatal(err)
		}

		// We should get a mostly-empty FastlyMeta for backend requests.
		meta, err := req.FastlyMeta()
		if err != nil {
			t.Fatalf("FastlyMeta: %v", err)
		}
		if meta == nil {
			t.Fatal("FastlyMeta() returned nil")
		}

		if got, want := meta.SandboxID, os.Getenv("FASTLY_TRACE_ID"); got != want {
			t.Errorf("FastlyMeta.SandboxID = %q, want: %q", got, want)
		}
		if got, want := meta.RequestID, ""; got != want {
			t.Errorf("FastlyMeta.RequestID = %q, want: %q", got, want)
		}
		if got, want := meta.SandboxRequests, 0; got != want {
			t.Errorf("FastlyMeta.SandboxRequests = %d, want: %d", got, want)
		}
	})

	t.Run("TLSClientCertificateInfo", func(t *testing.T) {
		req, err := fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
		if err != nil {
			t.Fatal(err)
		}

		ti, err := req.TLSClientCertificateInfo()
		if err != nil {
			t.Fatalf("TLSClientCertificateInfo: %v", err)
		}
		if ti == nil {
			t.Fatal("TLSClientCertificateInfo() returned nil")
		}

		if got, want := len(ti.RawClientCertificate), 0; got != want {
			t.Errorf("len(TLSClientCertificateInfo.RawClientCertificate) = %d, want: %d", got, want)
		}
	})

	t.Run("HandoffWebsocket", func(t *testing.T) {
		req, err := fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := req.HandoffWebsocket("self"), fsthttp.ErrHandoffNotSupported; got != want {
			t.Errorf("HandoffWebsocket() = %v, want: %v", got, want)
		}
	})

	t.Run("HandoffFanout", func(t *testing.T) {
		req, err := fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := req.HandoffFanout("self"), fsthttp.ErrHandoffNotSupported; got != want {
			t.Errorf("HandoffFanout() = %v, want: %v", got, want)
		}
	})
}
