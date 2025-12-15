//go:build wasip1 && !nofastlyhostcalls

package main

import (
	"context"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func TestSandboxReuse(t *testing.T) {
	// First request. Sandbox ID and request ID should match.
	req, err := fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := req.Send(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	sandboxID, requestID := resp.Header.Get("Sandbox-ID"), resp.Header.Get("Request-ID")
	sandboxRequests := resp.Header.Get("Sandbox-Requests")

	if sandboxID == "" || requestID == "" {
		t.Fatalf("Sandbox-ID and/or Request-ID are empty: %s, %s", sandboxID, requestID)
	}
	if sandboxID != requestID {
		t.Errorf("sandboxID = %s, requestID = %s; expected them to match", sandboxID, requestID)
	}
	if sandboxRequests != "1" {
		t.Errorf("sandboxRequests = %s; expected 1", sandboxRequests)
	}
	prevSandboxID := sandboxID

	// Second request.  This should reuse the sandbox, so the sandbox ID
	// should match the previous sandbox ID and the request ID should
	// not match.
	//
	// We also set a header to tell the server to not allow any more
	// requests on this sandbox.
	req, err = fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Fresh-Sandbox", "1")
	resp, err = req.Send(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	sandboxID, requestID = resp.Header.Get("Sandbox-ID"), resp.Header.Get("Request-ID")
	sandboxRequests = resp.Header.Get("Sandbox-Requests")

	if sandboxID != prevSandboxID {
		t.Errorf("sandboxID = %s, previous sandboxID = %s; expected them to match", sandboxID, sandboxID)
	}
	if sandboxID == requestID {
		t.Errorf("sandboxID = %s, requestID = %s; expected them to differ", sandboxID, requestID)
	}
	if sandboxRequests != "2" {
		t.Errorf("sandboxRequests = %s; expected 2", sandboxRequests)
	}
	prevSandboxID = sandboxID

	// Third request, we should have a new sandbox ID and it should match the request ID
	req, err = fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = req.Send(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	sandboxID, requestID = resp.Header.Get("Sandbox-ID"), resp.Header.Get("Request-ID")
	sandboxRequests = resp.Header.Get("Sandbox-Requests")

	if sandboxID == prevSandboxID {
		t.Errorf("sandboxID = %s, previous sandboxID = %s; expected them to differ", sandboxID, sandboxID)
	}
	if sandboxID != requestID {
		t.Errorf("sandboxID = %s, requestID = %s; expected them to match", sandboxID, requestID)
	}
	if sandboxRequests != "1" {
		t.Errorf("sandboxRequests = %s; expected 1", sandboxRequests)
	}
}
