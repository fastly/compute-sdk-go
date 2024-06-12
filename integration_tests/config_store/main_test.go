//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"bytes"
	"testing"

	"github.com/fastly/compute-sdk-go/configstore"
)

func TestConfigStore(t *testing.T) {
	d, err := configstore.Open("configstore")
	if err != nil {
		t.Fatal(err)
	}

	present, err := d.Has("missing-key")
	if err != nil {
		t.Fatal(err)
	}

	if present {
		t.Errorf("Has reported `true` for a missing key")
	}

	present, err = d.Has("empty-value")
	if err != nil {
		t.Fatal(err)
	}

	if !present {
		t.Errorf("Has reported `false` for a `empty-key`")
	}

	present, err = d.Has("twitter")
	if err != nil {
		t.Fatal(err)
	}

	if !present {
		t.Errorf("Missing key \"twitter\"")
	}

	tb, err := d.GetBytes("twitter")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := tb, "https://twitter.com/fastly"; !bytes.Equal(tb, []byte(want)) {
		t.Errorf("Body = %q, want %q", got, want)
	}

	twitter, err := d.Get("twitter")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := twitter, "https://twitter.com/fastly"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
