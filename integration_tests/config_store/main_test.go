//go:build tinygo.wasm && wasi && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/fastly/compute-sdk-go/configstore"
	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestConfigStore(t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, _ *fsthttp.Request) {
		d, err := configstore.Open("configstore")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		twitter, err := d.Get("twitter")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, twitter)
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

	if got, want := w.Body.String(), "https://twitter.com/fastly"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
