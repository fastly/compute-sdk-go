// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"reflect"
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

func TestHeaderApply(t *testing.T) {
	t.Parallel()

	h := NewHeader()
	h.Add("Host", "zombo.com")

	h2 := NewHeader()
	h2.Add("Host", "zombo2.com")

	h.Apply(h2)

	if got, want := h.Values("Host"), []string{"zombo2.com"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Host: got %q, want %q", got, want)
	}
}
