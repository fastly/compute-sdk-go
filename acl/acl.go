// Package acl provides access to Fastly ACLs.
//
// See the [Fastly ACL documentation] for details.
//
// [Fastly ACL documentation]: https://www.fastly.com/documentation/guides/concepts/edge-state/dynamic-config/#access-control-lists
package acl

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrNotFound indicates the requested ACL was not found.
	ErrNotFound = errors.New("acl: not found")

	// ErrInvalidHandle indicatest the ACL handle was invalid.
	ErrInvalidHandle = errors.New("acl: invalid handle")

	// ErrInvalidResponseBody indicates the looup response body was invalid.
	ErrInvalidResponseBody = errors.New("acl: invalid response body")

	// ErrInvalidArgument indicates the IP address was invalid.
	ErrInvalidArgument = errors.New("acl: invalid argument")

	// ErrNoContent indicates there was no entry for the provided IP address.
	ErrNoContent = errors.New("acl: no content")

	// ErrTooManyRequests indicates too many requests were made.
	ErrTooManyRequests = errors.New("acl: too many requests")

	// ErrUnexpected indicates an unexpected error occurred.
	ErrUnexpected = errors.New("acl: unexepected error")
)

// Handle is a handle for an ACL
type Handle struct {
	h *fastly.ACLHandle
}

// Response is an ACL lookup response
type Response struct {
	Prefix string // Matching prefix in CIDR notation
	Action string // Associated prefix's action
}

// Open returns a handle to the named ACL.
func Open(name string) (*Handle, error) {
	a, err := fastly.OpenACL(name)
	if err != nil {
		status, ok := fastly.IsFastlyError(err)
		switch {
		case ok && status == fastly.FastlyStatusNone:
			return nil, ErrNotFound
		case ok:
			return nil, fmt.Errorf("%w (%s)", ErrUnexpected, status)
		default:
			return nil, err
		}
	}

	return &Handle{h: a}, nil

}

// Lookup the given IP in the ACL and returns the response.  If no match was found, returns ErrNoContent.
func (h *Handle) Lookup(ip net.IP) (Response, error) {
	body, err := h.h.Lookup(ip)
	if err != nil {
		return Response{}, mapFastlyErr(err)
	}

	var r Response
	dec := json.NewDecoder(body)
	if err := dec.Decode(&r); err != nil {
		return Response{}, err
	}
	return r, nil
}

func mapFastlyErr(err error) error {
	// Is it a acl-specific error?
	if aclErr, ok := err.(fastly.ACLError); ok {
		switch aclErr {
		case fastly.ACLErrorUninitialized: // we really shouldn't be returning this
			return ErrUnexpected
		case fastly.ACLErrorOK:
			// Not an error; we shouldn't get here
			return fmt.Errorf("%w (%s)", ErrUnexpected, err)
		case fastly.ACLErrorNoContent:
			return ErrNoContent
		case fastly.ACLErrorTooManyRequests:
			return ErrTooManyRequests
		}
		return fmt.Errorf("%w (%s)", ErrUnexpected, err)
	}

	// Maybe it was a fastly error?
	status, ok := fastly.IsFastlyError(err)
	switch {
	case ok && status == fastly.FastlyStatusBadf:
		return ErrInvalidHandle
	case ok && status == fastly.FastlyStatusInval:
		return ErrInvalidArgument
	case ok:
		return fmt.Errorf("%w (%s)", ErrUnexpected, status)
	}

	// No idea; just return what we have.
	return err
}
