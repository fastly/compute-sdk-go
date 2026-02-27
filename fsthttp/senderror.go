// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"errors"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// SendError provides detailed information about backend request failures.
//
// Use errors.As() to extract the SendError from the error chain, then use the
// Cause() method to determine the specific error cause.
//
// Example usage:
//
//	resp, err := req.Send(ctx, "backend")
//	if err != nil {
//	    var se fsthttp.SendError
//	    if errors.As(err, &se) {
//	        switch se.Cause() {
//	        case fsthttp.SendErrorConnectionTimeout:
//	            log.Println("connection timed out")
//
//	        case fsthttp.SendErrorDNSError:
//	            log.Printf("DNS lookup failed: rcode=%d, info=%d", se.DNSErrorRCode(), se.DNSErrorInfoCode())
//
//	        case fsthttp.SendErrorTLSAlertReceived:
//	            log.Printf("TLS alert: %d (%s)", se.TLSAlertID(), se.TLSAlertDescription())
//	        }
//	    }
//	}
type SendError = fastly.SendErrorDetail

const (
	// SendErrorDNSTimeout indicates the system encountered a timeout when trying to
	// find an IP address for the backend hostname.
	SendErrorDNSTimeout = fastly.SendErrorDetailTagDNSTimeout

	// SendErrorDNSError indicates the system encountered a DNS error when trying to
	// find an IP address for the backend hostname.
	// Use DNSErrorRCode() and DNSErrorInfoCode() to get additional details.
	SendErrorDNSError = fastly.SendErrorDetailTagDNSError

	// SendErrorDestinationNotFound indicates the system cannot determine which backend
	// to use, or the specified backend was invalid.
	SendErrorDestinationNotFound = fastly.SendErrorDetailTagDestinationNotFound

	// SendErrorDestinationUnavailable indicates the system considers the backend to be
	// unavailable (e.g., recent attempts to communicate with it may have failed,
	// or a health check may indicate that it is down).
	SendErrorDestinationUnavailable = fastly.SendErrorDetailTagDestinationUnavailable

	// SendErrorDestinationIPUnroutable indicates the system cannot find a route to the
	// next-hop IP address.
	SendErrorDestinationIPUnroutable = fastly.SendErrorDetailTagDestinationIPUnroutable

	// SendErrorConnectionRefused indicates the system's connection to the backend was
	// refused.
	SendErrorConnectionRefused = fastly.SendErrorDetailTagConnectionRefused

	// SendErrorConnectionTerminated indicates the system's connection to the backend
	// was closed before a complete response was received.
	SendErrorConnectionTerminated = fastly.SendErrorDetailTagConnectionTerminated

	// SendErrorConnectionTimeout indicates the system's attempt to open a connection
	// to the backend timed out.
	SendErrorConnectionTimeout = fastly.SendErrorDetailTagConnectionTimeout

	// SendErrorConnectionLimitReached indicates the system is configured to limit the
	// number of connections it has to the backend, and that limit has been reached.
	SendErrorConnectionLimitReached = fastly.SendErrorDetailTagConnectionLimitReached

	// SendErrorTLSCertificateError indicates the system encountered an error when
	// verifying the certificate presented by the backend.
	SendErrorTLSCertificateError = fastly.SendErrorDetailTagTLSCertificateError

	// SendErrorTLSConfigurationError indicates the system encountered an error with
	// the backend TLS configuration.
	SendErrorTLSConfigurationError = fastly.SendErrorDetailTagTLSConfigurationError

	// SendErrorTLSAlertReceived indicates the system received a TLS alert from the
	// backend. Use TLSAlertID() and TLSAlertDescription() to get the specific alert.
	SendErrorTLSAlertReceived = fastly.SendErrorDetailTagTLSAlertReceived

	// SendErrorTLSProtocolError indicates the system encountered a TLS error when
	// communicating with the backend, either during the handshake or afterwards.
	SendErrorTLSProtocolError = fastly.SendErrorDetailTagTLSProtocolError

	// SendErrorHTTPIncompleteResponse indicates the system received an incomplete
	// response to the request from the backend.
	SendErrorHTTPIncompleteResponse = fastly.SendErrorDetailTagHTTPIncompleteResponse

	// SendErrorHTTPResponseHeaderSectionTooLarge indicates the system received a
	// response to the request whose header section was considered too large.
	// For specific limits, visit the [Compute resource limits documentation].
	//
	// [Compute resource limits documentation]: https://docs.fastly.com/products/compute-resource-limits
	SendErrorHTTPResponseHeaderSectionTooLarge = fastly.SendErrorDetailTagHTTPResponseHeaderSectionTooLarge

	// SendErrorHTTPResponseBodyTooLarge indicates the system received a response to
	// the request whose body was considered too large.
	// For specific limits, visit the [Compute resource limits documentation].
	//
	// [Compute resource limits documentation]: https://docs.fastly.com/products/compute-resource-limits
	SendErrorHTTPResponseBodyTooLarge = fastly.SendErrorDetailTagHTTPResponseBodyTooLarge

	// SendErrorHTTPResponseTimeout indicates the system reached a configured time
	// limit waiting for the complete response.  The limit is configured on the backend.
	SendErrorHTTPResponseTimeout = fastly.SendErrorDetailTagHTTPResponseTimeout

	// SendErrorHTTPResponseStatusInvalid indicates the system received a response to
	// the request whose status code or reason phrase was invalid.
	SendErrorHTTPResponseStatusInvalid = fastly.SendErrorDetailTagHTTPResponseStatusInvalid

	// SendErrorHTTPUpgradeFailed indicates the process of negotiating an upgrade of
	// the HTTP version between the system and the backend failed.
	SendErrorHTTPUpgradeFailed = fastly.SendErrorDetailTagHTTPUpgradeFailed

	// SendErrorHTTPProtocolError indicates the system encountered an HTTP protocol
	// error when communicating with the backend. This error will only be used when a more
	// specific one is not defined.
	SendErrorHTTPProtocolError = fastly.SendErrorDetailTagHTTPProtocolError

	// SendErrorHTTPRequestCacheKeyInvalid indicates an invalid Fastly cache key was
	// provided for the request.
	SendErrorHTTPRequestCacheKeyInvalid = fastly.SendErrorDetailTagHTTPRequestCacheKeyInvalid

	// SendErrorHTTPRequestURIInvalid indicates an invalid URI was provided for the
	// request.
	SendErrorHTTPRequestURIInvalid = fastly.SendErrorDetailTagHTTPRequestURIInvalid

	// SendErrorInternalError indicates the system encountered an unexpected internal
	// error.
	SendErrorInternalError = fastly.SendErrorDetailTagInternalError
)

var (
	ErrRequestCollapse = errors.New("error during request collapse")
)
