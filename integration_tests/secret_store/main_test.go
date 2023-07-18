//go:build tinygo.wasm && wasi && !nofastlyhostcalls

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
	"github.com/fastly/compute-sdk-go/secretstore"
)

func TestSecretStore(t *testing.T) {
	handler := func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		st, err := secretstore.Open("phrases")
		switch {
		case errors.Is(err, secretstore.ErrSecretStoreNotFound):
			fsthttp.Error(w, err.Error(), fsthttp.StatusNotFound)
			return
		case err != nil:
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		s, err := st.Get("my_phrase")
		switch {
		case errors.Is(err, secretstore.ErrSecretNotFound):
			fsthttp.Error(w, err.Error(), fsthttp.StatusNotFound)
			return
		case err != nil:
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		v, err := s.Plaintext()
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		w.Write(v)
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

	if got, want := w.Body.String(), "sssh! don't tell anyone!"; got != want {
		t.Errorf("Body = %q, want %q", got, want)
	}
}
