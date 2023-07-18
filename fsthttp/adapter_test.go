// This test file is in its own test package to avoid a circular
// dependency between fsthttp and fsttest.

package fsthttp_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestAdapter(t *testing.T) {
	t.Parallel()

	hh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fr := fsthttp.RequestFromContext(r.Context())
		if fr == nil {
			http.Error(w, "no fsthttp.Request in context", http.StatusInternalServerError)
			return
		}

		fw := fsthttp.ResponseWriterFromContext(r.Context())
		if fw == nil {
			http.Error(w, "no fsthttp.ResponseWriter in context", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusTeapot)
		fmt.Fprintln(w, "Hello, client")
	})

	r, err := fsthttp.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := fsttest.NewRecorder()

	fsthttp.Adapt(hh).ServeHTTP(context.Background(), w, r)

	if want, got := fsthttp.StatusTeapot, w.Code; want != got {
		t.Errorf("want code %d, got %d", want, got)
	}

	if want, got := "Hello, client\n", w.Body.String(); want != got {
		t.Errorf("want body %q, got %q", want, got)
	}
}
