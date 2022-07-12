// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"testing"
)

func TestHeaderBasics(t *testing.T) {
	t.Parallel()

	h := NewHeader()

	h.Add("Host", "zombo.com")
	if want, have := "zombo.com", h.Get("host"); want != have {
		t.Errorf("Host: want %q, have %q", want, have)
	}
}
