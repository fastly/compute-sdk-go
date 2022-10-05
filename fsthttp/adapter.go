package fsthttp

import (
	"context"
	"net/http"
)

// responseWriterAdapter is an implementation of http.ResponseWriter on
// top of fsthttp.ResponseWriter.  It is necessary because the Header
// types are different, despite being otherwise compatible.
type responseWriterAdapter struct {
	w ResponseWriter
}

func (w *responseWriterAdapter) Header() http.Header {
	return http.Header(w.w.Header())
}

func (w *responseWriterAdapter) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

func (w *responseWriterAdapter) WriteHeader(status int) {
	w.w.WriteHeader(status)
}

// Adapt allows an http.Handler to be used as an fsthttp.Handler.
//
// Because the Request and ResponseWriter types are not exactly the same
// as ones in net/http, helper accessor functions exist to extract the
// fsthttp values from the request context.
func Adapt(h http.Handler) Handler {
	return HandlerFunc(func(ctx context.Context, w ResponseWriter, r *Request) {
		ctx = contextWithRequest(ctx, r)
		ctx = contextWithResponseWriter(ctx, w)

		hw := &responseWriterAdapter{w: w}

		hr, err := http.NewRequestWithContext(ctx, r.Method, r.URL.String(), r.Body)
		if err != nil {
			w.WriteHeader(StatusInternalServerError)
			return
		}
		hr.Proto = r.Proto
		hr.ProtoMajor = r.ProtoMajor
		hr.ProtoMinor = r.ProtoMinor
		hr.Header = http.Header(r.Header.Clone())
		hr.Host = r.Host
		hr.RemoteAddr = r.RemoteAddr
		hr.ContentLength = -1

		// TODO: Translate some of fsthttp.TLSInfo into
		// tls.ConnectionState.
		//
		// The protocol version and chosen cipher are available but
		// provided via the ABI as strings, which we would need to
		// convert back into integer values.
		//
		// The raw ClientHello is provided, so we could use
		// golang.org/x/crypto/cryptobyte to parse it.  But
		// server-chosen properties of the connection (cipher, ALPN,
		// client certificate, etc.) would need to be provided by the
		// ABI.

		h.ServeHTTP(hw, hr)
	})
}
