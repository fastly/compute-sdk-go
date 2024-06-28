package fsttest

import (
	"bytes"
	"fmt"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// ResponseRecorder is an implementation of fsthttp.ResponseWriter that
// records its mutations for later inspection in tests.
type ResponseRecorder struct {
	Code        int
	HeaderMap   fsthttp.Header
	Body        *bytes.Buffer
	headersDone bool
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
	if !r.headersDone {
		return r.HeaderMap
	}
	// Once the send the headers, return a copy so any changes
	// are discarded.
	return r.HeaderMap.Clone()
}

// WriteHeader records the response code.
func (r *ResponseRecorder) WriteHeader(code int) {
	if !r.headersDone {
		r.Code = code
		r.headersDone = true
	}
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

// Append records the response body.  The data is written to the Body
// field of the ResponseRecorder.
func (r *ResponseRecorder) Append(other io.ReadCloser) error {
	// do the same type check as the real implementation
	_, ok := other.(*fastly.HTTPBody)
	if !ok {
		return fmt.Errorf("non-Response Body passed to ResponseWriter.Append")
	}
	// the real implementation makes a host call to do the body append
	// without a real body handle to write to, we'll just use io.Copy to the same
	// effect
	_, err := io.Copy(r, other)
	return err
}
