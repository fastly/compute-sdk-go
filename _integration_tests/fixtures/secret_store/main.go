package main

import (
	"context"
	"errors"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/secretstore"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
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
	})
}
