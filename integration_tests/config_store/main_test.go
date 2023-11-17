//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"testing"

	"github.com/fastly/compute-sdk-go/configstore"
)

func TestConfigStore(t *testing.T) {
	d, err := configstore.Open("configstore")
	if err != nil {
		t.Fatal(err)
	}

	twitter, err := d.Get("twitter")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := twitter, "https://twitter.com/fastly"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
