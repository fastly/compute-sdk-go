//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func NewBackendOptions() *fsthttp.BackendOptions {
	return &fsthttp.BackendOptions{}
}

func TestDynamicBackend(t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		b, err := fsthttp.RegisterDynamicBackend(
			"dynamic",
			"compute-sdk-test-backend.edgecompute.app",
			NewBackendOptions().UseSSL(true),
		)
		if err != nil {
			t.Errorf("RegisterDynamicBackend: %v", err)
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Printf("%+v\n", b)

		if !b.IsDynamic() {
			t.Errorf("IsDynamic() = false, want true")
			fsthttp.Error(w, "IsDynamic() = false, want true", fsthttp.StatusInternalServerError)
			return
		}

		if !b.IsSSL() {
			t.Errorf("IsSSL() = false, want true")
			fsthttp.Error(w, "IsSSL() = false, want true", fsthttp.StatusInternalServerError)
			return
		}

		health, err := b.Health()
		if err != nil {
			t.Errorf("Health: %v", err)
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		// Viceroy doesn't support health checks, so the status will always be unknown
		if health != fsthttp.BackendHealthUnknown {
			t.Errorf("Health = %v, want %v", health, fsthttp.BackendHealthUnknown)
			fsthttp.Error(w, fmt.Sprintf("Health = %v, want %v", health, fsthttp.BackendHealthUnknown), fsthttp.StatusInternalServerError)
			return
		}

		req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/", nil)
		if err != nil {
			t.Errorf("NewRequest: %v", err)
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		req.CacheOptions.Pass = true

		// Send to our newly-registered dynamic backend
		resp, err := req.Send(ctx, "dynamic")
		if err != nil {
			t.Errorf("Send: %v", err)
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		w.Header().Reset(resp.Header.Clone())
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			t.Errorf("Copy: %v", err)
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}
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

	if got, want := w.Body.String(), "Compute SDK Test Backend"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
