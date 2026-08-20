//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func TestDynamicBackend(t *testing.T) {
	b, err := fsthttp.RegisterDynamicBackend(
		"dynamic",
		"compute-sdk-test-backend.edgecompute.app",
		fsthttp.NewBackendOptions().UseSSL(true),
	)
	if err != nil {
		t.Fatalf("RegisterDynamicBackend: %v", err)
	}

	if !b.IsDynamic() {
		t.Errorf("IsDynamic() = false, want true")
	}

	if !b.IsSSL() {
		t.Errorf("IsSSL() = false, want true")
	}

	health, err := b.Health()
	if err != nil {
		t.Fatalf("Health: %v", err)
	}

	// Viceroy doesn't support health checks, so the status will always be unknown
	if health != fsthttp.BackendHealthUnknown {
		t.Errorf("Health = %v, want %v", health, fsthttp.BackendHealthUnknown)
	}

	req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	req.CacheOptions.Pass = true

	// Send to our newly-registered dynamic backend
	resp, err := req.Send(context.Background(), "dynamic")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, fsthttp.StatusOK; got != want {
		t.Errorf("StatusCode = %d, want %d", got, want)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if got, want := string(body), "Compute SDK Test Backend"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}

func TestDynamicBackendNilOptions(t *testing.T) {
	b, err := fsthttp.RegisterDynamicBackend(
		"dynamic-nil-opts",
		"compute-sdk-test-backend.edgecompute.app",
		nil,
	)
	if err != nil {
		t.Fatalf("RegisterDynamicBackend: %v", err)
	}

	if got, want := b.IsDynamic(), true; got != want {
		t.Errorf("IsDynamic() = %v, want %v", got, want)
	}
}

func TestDynamicBackendHealthcheck(t *testing.T) {
	hc := fsthttp.NewBackendHealthcheck("compute-sdk-test-backend.edgecompute.app").
		Method("GET").
		Path("/").
		Status(200).
		Window(10).
		Threshold(5).
		Initial(6).
		Interval(20 * time.Second).
		Timeout(10 * time.Second)

	b, err := fsthttp.RegisterDynamicBackend(
		"dynamic-healthcheck",
		"compute-sdk-test-backend.edgecompute.app",
		fsthttp.NewBackendOptions().Healthcheck(hc),
	)
	if err != nil {
		t.Errorf("RegisterDynamicBackend: %v", err)
	}

	if got, want := b.IsDynamic(), true; got != want {
		t.Errorf("IsDynamic() = %v, want %v", got, want)
	}

	// Viceroy doesn't support health checks, so the status will always be unknown
	health, err := b.Health()
	if err != nil {
		t.Errorf("Health: %v", err)
	}

	if health != fsthttp.BackendHealthUnknown {
		t.Errorf("Health = %v, want %v", health, fsthttp.BackendHealthUnknown)
	}
}

func TestOriginHealth(t *testing.T) {
	b, err := fsthttp.BackendFromName("healthy")
	if err != nil {
		t.Fatalf("BackendFromName: %v", err)
	}

	health, err := b.Health()
	if err != nil {
		t.Fatalf("Health: %v", err)
	}

	if health != fsthttp.BackendHealthHealthy {
		t.Errorf("Health = %v, want %v", health, fsthttp.BackendHealthHealthy)
	}

	b, err = fsthttp.BackendFromName("unhealthy")
	if err != nil {
		t.Fatalf("BackendFromName: %v", err)
	}

	health, err = b.Health()
	if err != nil {
		t.Fatalf("Health: %v", err)
	}

	if health != fsthttp.BackendHealthUnhealthy {
		t.Errorf("Health = %v, want %v", health, fsthttp.BackendHealthUnhealthy)
	}
}
