package fsthttp

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
)

// A poor replacement for httptest.ResponseRecorder
type ResponseRecorder struct {
	Code      int
	HeaderMap Header
	Body      *bytes.Buffer
}

func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		Code:      StatusOK,
		HeaderMap: make(Header),
		Body:      &bytes.Buffer{},
	}
}

func (r *ResponseRecorder) Header() Header {
	return r.HeaderMap
}

func (r *ResponseRecorder) WriteHeader(code int) {
	r.Code = code
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	return r.Body.Write(b)
}

func (r *ResponseRecorder) Close() error {
	return nil
}

func (r *ResponseRecorder) SetManualFramingMode(v bool) {}

func TestAdapter(t *testing.T) {
	t.Parallel()

	hh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fr := RequestFromContext(r.Context())
		if fr == nil {
			http.Error(w, "no fsthttp.Request in context", http.StatusInternalServerError)
			return
		}

		fw := ResponseWriterFromContext(r.Context())
		if fw == nil {
			http.Error(w, "no fsthttp.ResponseWriter in context", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusTeapot)
		fmt.Fprintln(w, "Hello, client")
	})

	r, err := NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := NewRecorder()

	Adapt(hh).ServeHTTP(context.Background(), w, r)

	if want, got := StatusTeapot, w.Code; want != got {
		t.Errorf("want code %d, got %d", want, got)
	}

	if want, got := "Hello, client\n", w.Body.String(); want != got {
		t.Errorf("want body %q, got %q", want, got)
	}
}
