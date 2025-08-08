//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
	"github.com/fastly/compute-sdk-go/shielding"
)

func TestShielding(t *testing.T) {
	handler := func(_ context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		name := r.URL.Query().Get("shield")

		shield, err := shielding.ShieldFromName(name)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Name=%v Target=%v SSL=%v", shield.Name(), shield.Target(), shield.SSLTarget())
	}

	r, err := fsthttp.NewRequest("GET", "/?shield=bfi-wa-us", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	w := fsttest.NewRecorder()

	handler(context.Background(), w, r)

	if got, want := w.Code, fsthttp.StatusOK; got != want {
		t.Errorf("Code = %d, want %d", got, want)
	}

	if got, want := w.Body.String(), "Name=bfi-wa-us Target=http://localhost/ SSL=https://localhost/"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
