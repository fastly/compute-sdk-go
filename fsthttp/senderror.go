// Copyright 2022 Fastly, Inc.

package fsthttp

import "github.com/fastly/compute-sdk-go/internal/abi/fastly"

// SendError provides detailed information about backend request failures.
//
// Use errors.Is() with the sentinel error variables to check for specific error values.
// Use errors.As() to extract the SendError and access detailed error information if needed.
//
// Example usage:
//
//	resp, err := req.Send(ctx, "backend")
//	if err != nil {
//	    // Check for specific error values using errors.Is()
//	    if errors.Is(err, fsthttp.ErrConnectionTimeout) {
//	        log.Println("connection timed out")
//	        return
//	    }
//
//	    // For DNS errors, extract details using errors.As()
//	    if errors.Is(err, fsthttp.ErrDNSError) {
//	        var se fsthttp.SendError
//	        errors.As(err, &se)
//	        rcode := se.DNSErrorRCode()
//	        infoCode := se.DNSErrorInfoCode()
//	        log.Printf("DNS lookup failed: rcode=%d, info=%d", rcode, infoCode)
//	        return
//	    }
//
//	    // For TLS alert errors, extract alert details
//	    if errors.Is(err, fsthttp.ErrTLSAlertReceived) {
//	        var se fsthttp.SendError
//	        errors.As(err, &se)
//	        log.Printf("TLS alert: %s (id=%d)", se.TLSAlertDescription(), se.TLSAlertID())
//	        return
//	    }
//	}
type SendError = fastly.SendErrorDetail

// Sentinel errors for backend request failures.
// Use these with errors.Is() to check for specific error values.
var (
	// ErrDNSTimeout indicates the system encountered a timeout when trying to
	// find an IP address for the backend hostname.
	ErrDNSTimeout = fastly.SendErrorDNSTimeout

	// ErrDNSError indicates the system encountered a DNS error when trying to
	// find an IP address for the backend hostname.
	// Use DNSErrorRCode() and DNSErrorInfoCode() to get additional details.
	ErrDNSError = fastly.SendErrorDNSError

	// ErrDestinationNotFound indicates the system cannot determine which backend
	// to use, or the specified backend was invalid.
	ErrDestinationNotFound = fastly.SendErrorDestinationNotFound

	// ErrDestinationUnavailable indicates the system considers the backend to be
	// unavailable (e.g., recent attempts to communicate with it may have failed,
	// or a health check may indicate that it is down).
	ErrDestinationUnavailable = fastly.SendErrorDestinationUnavailable

	// ErrDestinationIPUnroutable indicates the system cannot find a route to the
	// next-hop IP address.
	ErrDestinationIPUnroutable = fastly.SendErrorDestinationIPUnroutable

	// ErrConnectionRefused indicates the system's connection to the backend was
	// refused.
	ErrConnectionRefused = fastly.SendErrorConnectionRefused

	// ErrConnectionTerminated indicates the system's connection to the backend
	// was closed before a complete response was received.
	ErrConnectionTerminated = fastly.SendErrorConnectionTerminated

	// ErrConnectionTimeout indicates the system's attempt to open a connection
	// to the backend timed out.
	ErrConnectionTimeout = fastly.SendErrorConnectionTimeout

	// ErrConnectionLimitReached indicates the system is configured to limit the
	// number of connections it has to the backend, and that limit has been
	// reached.
	ErrConnectionLimitReached = fastly.SendErrorConnectionLimitReached

	// ErrTLSCertificateError indicates the system encountered an error when
	// verifying the certificate presented by the backend.
	ErrTLSCertificateError = fastly.SendErrorTLSCertificateError

	// ErrTLSConfigurationError indicates the system encountered an error with
	// the backend TLS configuration.
	ErrTLSConfigurationError = fastly.SendErrorTLSConfigurationError

	// ErrTLSAlertReceived indicates the system received a TLS alert from the
	// backend. Use TLSAlertID() and TLSAlertDescription() to get the specific alert.
	ErrTLSAlertReceived = fastly.SendErrorTLSAlertReceived

	// ErrTLSProtocolError indicates the system encountered a TLS error when
	// communicating with the backend, either during the handshake or afterwards.
	ErrTLSProtocolError = fastly.SendErrorTLSProtocolError

	// ErrHTTPIncompleteResponse indicates the system received an incomplete
	// response to the request from the backend.
	ErrHTTPIncompleteResponse = fastly.SendErrorHTTPIncompleteResponse

	// ErrHTTPResponseHeaderSectionTooLarge indicates the system received a
	// response to the request whose header section was considered too large.
	// For specific limits, visit the [Compute resource limits documentation].
	//
	// [Compute resource limits documentation]: https://docs.fastly.com/products/compute-resource-limits
	ErrHTTPResponseHeaderSectionTooLarge = fastly.SendErrorHTTPResponseHeaderSectionTooLarge

	// ErrHTTPResponseBodyTooLarge indicates the system received a response to
	// the request whose body was considered too large.
	// For specific limits, visit the [Compute resource limits documentation].
	//
	// [Compute resource limits documentation]: https://docs.fastly.com/products/compute-resource-limits
	ErrHTTPResponseBodyTooLarge = fastly.SendErrorHTTPResponseBodyTooLarge

	// ErrHTTPResponseTimeout indicates the system reached a configured time
	// limit waiting for the complete response.  The limit is configured on the
	// backend.
	ErrHTTPResponseTimeout = fastly.SendErrorHTTPResponseTimeout

	// ErrHTTPResponseStatusInvalid indicates the system received a response to
	// the request whose status code or reason phrase was invalid.
	ErrHTTPResponseStatusInvalid = fastly.SendErrorHTTPResponseStatusInvalid

	// ErrHTTPUpgradeFailed indicates the process of negotiating an upgrade of
	// the HTTP version between the system and the backend failed.
	ErrHTTPUpgradeFailed = fastly.SendErrorHTTPUpgradeFailed

	// ErrHTTPProtocolError indicates the system encountered an HTTP protocol
	// error when communicating with the backend. This error will only be used
	// when a more specific one is not defined.
	ErrHTTPProtocolError = fastly.SendErrorHTTPProtocolError

	// ErrHTTPRequestCacheKeyInvalid indicates an invalid Fastly cache key was provided
	// for the request.
	ErrHTTPRequestCacheKeyInvalid = fastly.SendErrorHTTPRequestCacheKeyInvalid

	// ErrHTTPRequestURIInvalid indicates an invalid URI was provided for the
	// request.
	ErrHTTPRequestURIInvalid = fastly.SendErrorHTTPRequestURIInvalid

	// ErrInternalError indicates the system encountered an unexpected internal
	// error.
	ErrInternalError = fastly.SendErrorInternalError
)
