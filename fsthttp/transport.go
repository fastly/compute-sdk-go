package fsthttp

import (
	"net/http"
	"strings"
)

// Transport is an http.RoundTripper implementation for backend requests
// on Compute@Edge.
//
// Compute@Edge requests must be made to a pre-configured named backend.
// A default backend is set when the transport is created, but
// additional backends can be added with the AddBackend method.
//
// This is primarily intended to adapt existing code which uses
// configurable http.Client instances to work on Compute@Edge.  Using an
// http.Client pulls in substantially more code, resulting in slower
// compile times and larger binaries.  For this reason, we recommend new
// code use the fsthttp.Request type and its Send() method directly
// whenever possible.
type Transport struct {
	defaultBackend string
	backends       map[string]string

	// Request is an optional callback invoked before the request is
	// sent to the backend.  It allows callers to set
	// fsthttp.Request-specific fields, such as cache control options.
	Request func(req *Request) error
}

// NewTransport creates a new Transport instance with the given default
// backend.
func NewTransport(backend string) *Transport {
	return &Transport{
		defaultBackend: backend,
		backends:       make(map[string]string),
	}
}

// AddBackend adds a new backend to the transport.
func (t *Transport) AddBackend(name, host string) {
	t.backends[strings.ToLower(host)] = name
}

func (t *Transport) getBackend(host string) string {
	if backend, ok := t.backends[strings.ToLower(host)]; ok {
		return backend
	}
	return t.defaultBackend
}

// RoundTrip implements the http.RoundTripper interface.
//
// The provided http.Request is adapted into an fsthttp.Request. If the
// Transport's Request callback field is set, it is invoked so that the
// fsthttp.Request can be modified before it is sent.  The request is
// then sent to the backend matching the host in the URL.  The resulting
// fsthttp.Response is adapted into an http.Response and returned.
//
// The http.Response's Request field contains a context from which the
// original fsthttp.Request and fsthttp.Response can be extracted using
// fsthttp.RequestFromContext and fsthttp.ResponseFromContext.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	freq, err := NewRequest(req.Method, req.URL.String(), req.Body)
	if err != nil {
		return nil, err
	}
	freq.Header = Header(req.Header.Clone())

	if t.Request != nil {
		if err := t.Request(freq); err != nil {
			return nil, err
		}
	}

	fresp, err := freq.Send(req.Context(), t.getBackend(req.URL.Host))
	if err != nil {
		return nil, err
	}

	ctx := contextWithRequest(req.Context(), freq)
	ctx = contextWithResponse(ctx, fresp)

	resp := &http.Response{
		Request:    req.WithContext(ctx),
		StatusCode: fresp.StatusCode,
		Header:     http.Header(fresp.Header.Clone()),
		Body:       fresp.Body,
	}

	return resp, nil
}
