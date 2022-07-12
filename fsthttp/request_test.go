// Copyright 2022 Fastly, Inc.

package fsthttp

import "testing"

// TestRequestHost validates a Host field is set on the Request type.
func TestRequestHost(t *testing.T) {
	t.Parallel()

	uri := "http://example.com:8080/"
	want := "example.com:8080"

	r, err := NewRequest("GET", uri, nil)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if want, have := want, r.Host; want != have {
		t.Errorf("Host: want %q, have %q", want, have)
	}

	if want, have := want, r.URL.Host; want != have {
		t.Errorf("Host: want %q, have %q", want, have)
	}
}
