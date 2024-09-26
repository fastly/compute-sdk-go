package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/secretstore"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		v, err := secretstore.Plaintext("example_secretstore", "my_secret")
		switch {
		case errors.Is(err, secretstore.ErrSecretStoreNotFound) || errors.Is(err, secretstore.ErrSecretNotFound):
			fsthttp.Error(w, err.Error(), fsthttp.StatusNotFound)
			return
		case err != nil:
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		fmt.Fprintf(w, "secret value: %q", v)
	})
}
