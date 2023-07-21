package fsttest

import (
	"bytes"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

// ResponseRecorder is an implementation of fsthttp.ResponseWriter that
// records its mutations for later inspection in tests.
type ResponseRecorder struct {
	Code      int
	HeaderMap fsthttp.Header
	Body      *bytes.Buffer
}

// NewRecorder returns an initialized ResponseRecorder.
func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		Code:      fsthttp.StatusOK,
		HeaderMap: make(fsthttp.Header),
		Body:      &bytes.Buffer{},
	}
}

// Header returns the response headers to mutate within a handler.
func (r *ResponseRecorder) Header() fsthttp.Header {
	return r.HeaderMap
}

// WriteHeader records the response code.
func (r *ResponseRecorder) WriteHeader(code int) {
	r.Code = code
}

// Write records the response body.  The data is written to the Body
// field of the ResponseRecorder.
func (r *ResponseRecorder) Write(b []byte) (int, error) {
	return r.Body.Write(b)
}

// Close is a no-op on ResponseRecorder.  It exists to satisfy the
// fsthttp.ResponseWriter interface.
func (r *ResponseRecorder) Close() error {
	return nil
}

// SetManualFramingMode is a no-op on ResponseRecorder.  It exists to
// satisfy the fsthttp.ResponseWriter interface.
func (r *ResponseRecorder) SetManualFramingMode(v bool) {}
