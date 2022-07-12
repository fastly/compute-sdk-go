// Copyright 2022 Fastly, Inc.

package rtlog

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

// Endpoint represents a real-time logging endpoint.
type Endpoint struct {
	abiEndpoint *fastly.LogEndpoint
	err         error
}

// Open returns an endpoint corresponding to the given name. Names are case
// sensitive. Calling Open with a name that doesn't correspond to any logging
// endpoint available in your service will still return a usable endpoint, and
// writes to that endpoint will succeed. Refer to your service dashboard to
// diagnose missing log events.
func Open(name string) *Endpoint {
	e, err := fastly.GetLogEndpoint(name)
	return &Endpoint{e, err}
}

// Write implements io.Writer, writing len(p) bytes from p to the endpoint.
// Returns the number of bytes written, and any error encountered.
// By contract, if n < len(p), the returned error will be non-nil.
//
// Each call to Write produces a single log event.
func (e *Endpoint) Write(p []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}

	return e.abiEndpoint.Write(p)
}
