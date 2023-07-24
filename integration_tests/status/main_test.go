//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestStatus(t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		fsthttp.Error(w, "Unauthorized", fsthttp.StatusUnauthorized)
	}

	r, err := fsthttp.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	w := fsttest.NewRecorder()

	handler(context.Background(), w, r)

	if got, want := w.Code, fsthttp.StatusUnauthorized; got != want {
		t.Errorf("Code = %d, want %d", got, want)
	}
}
