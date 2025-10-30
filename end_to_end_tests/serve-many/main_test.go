//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

package main

import (
	"context"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func TestSessionReuse(t *testing.T) {
	// First request.  Session ID and request ID should match.
	req, err := fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := req.Send(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	sessionID, requestID := resp.Header.Get("Session-ID"), resp.Header.Get("Request-ID")

	if sessionID == "" || requestID == "" {
		t.Fatalf("Session-ID and/or Request-ID are empty: %s, %s", sessionID, requestID)
	}
	if sessionID != requestID {
		t.Errorf("sessionID = %s, requestID = %s; expected them to match", sessionID, requestID)
	}
	prevSessionID := sessionID

	// Second request.  This should reuse the session, so the session ID
	// should match the previous session ID and the request ID should
	// not match.
	//
	// We also set a header to tell the server to not allow any more
	// requests on this session.
	req, err = fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Close-Session", "1")
	resp, err = req.Send(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	sessionID, requestID = resp.Header.Get("Session-ID"), resp.Header.Get("Request-ID")

	if sessionID != prevSessionID {
		t.Errorf("sessionID = %s, previous sessionID = %s; expected them to match", sessionID, sessionID)
	}
	if sessionID == requestID {
		t.Errorf("sessionID = %s, requestID = %s; expected them to differ", sessionID, requestID)
	}
	prevSessionID = sessionID

	// Third request, we should have a new session ID and it should match the request ID
	req, err = fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err = req.Send(context.Background(), "self")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	sessionID, requestID = resp.Header.Get("Session-ID"), resp.Header.Get("Request-ID")

	if sessionID == prevSessionID {
		t.Errorf("sessionID = %s, previous sessionID = %s; expected them to differ", sessionID, sessionID)
	}
	if sessionID != requestID {
		t.Errorf("sessionID = %s, requestID = %s; expected them to match", sessionID, requestID)
	}
}
