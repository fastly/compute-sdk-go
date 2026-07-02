package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/secretstore"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		v, err := secretstore.Plaintext("example_secretstore", "my_secret")
		switch {
		case errors.Is(err, secretstore.ErrSecretStoreNotFound) || errors.Is(err, secretstore.ErrSecretNotFound):
			log.Println("error looking up secret:", err)
			fsthttp.Error(w, fsthttp.StatusText(fsthttp.StatusNotFound), fsthttp.StatusNotFound)
			return
		case err != nil:
			log.Println("error looking up secret:", err)
			fsthttp.Error(w, fsthttp.StatusText(fsthttp.StatusBadGateway), fsthttp.StatusBadGateway)
			return
		}

		// SECURITY: We're writing the decrypted secret back to the user.  DON'T DO THIS!
		// In reality this would be an API key or equivalent added to
		// an outgoing HTTP request header, or perhaps a key used to
		// decrypt or verify a request header or body.
		fmt.Fprintf(w, "secret value: %q", v)
	})
}
