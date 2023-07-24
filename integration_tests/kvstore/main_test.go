//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2023 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
	"github.com/fastly/compute-sdk-go/kvstore"
)

func TestKVStore(t *testing.T) {
	handler := func(ctx context.Context, res fsthttp.ResponseWriter, req *fsthttp.Request) {
		store, err := kvstore.Open("example-test-kv-store")
		if err != nil {
			fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		switch req.URL.Path {
		case "/lookup":
			{
				hello, err := store.Lookup("hello")
				if err != nil {
					fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
					return
				}

				fmt.Fprint(res, hello.String())
			}
		case "/insert":
			{
				err := store.Insert("animal", strings.NewReader("cat"))
				if err != nil {
					fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
					return
				}

				animal, err := store.Lookup("animal")
				if err != nil {
					fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
					return
				}

				fmt.Fprint(res, animal.String())
			}
		}
	}

	testcases := []struct {
		name     string
		wantBody string
	}{
		{"lookup", "world"},
		{"insert", "cat"},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := fsthttp.NewRequest("GET", "/"+tc.name, nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			w := fsttest.NewRecorder()

			handler(context.Background(), w, r)

			if got, want := w.Code, fsthttp.StatusOK; got != want {
				t.Errorf("Code = %d, want %d", got, want)
			}

			if got, want := w.Body.String(), tc.wantBody; got != want {
				t.Errorf("Body = %q, want %q", got, want)
			}
		})
	}
}
