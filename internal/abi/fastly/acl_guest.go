//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2024 Fastly, Inc.

package fastly

import (
	"io"
	"net"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
//
//    (@interface func (export "open")
//        (param $name string)
//        (result $err (expected $acl_handle (error $fastly_status)))
//    )

//go:wasmimport fastly_acl open
//go:noescape
func fastlyACLOpen(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	h prim.Pointer[aclHandle],
) FastlyStatus

// ACL is a handle to the ACL subsystem.
type ACLHandle struct {
	h aclHandle
}

// OpenACL returns a handle to the named ACL set.
func OpenACL(name string) (*ACLHandle, error) {
	var acl ACLHandle

	nameBuffer := prim.NewReadBufferFromString(name).Wstring()

	if err := fastlyACLOpen(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&acl.h),
	).toError(); err != nil {
		return nil, err
	}

	return &acl, nil
}

// witx:
//
//    (@interface func (export "lookup")
//        (param $acl $acl_handle)
//        (param $ip_octets (@witx const_pointer (@witx char8)))
//        (param $ip_len (@witx usize))
//        (param $body_handle_out (@witx pointer $body_handle))
//        (param $acl_error_out (@witx pointer $acl_error))
//        (result $err (expected (error $fastly_status)))
//    )

//go:wasmimport fastly_acl lookup
//go:noescape
func fastlyACLLookup(
	h aclHandle,
	ipData prim.Pointer[prim.U8], ipLen prim.Usize,
	b prim.Pointer[bodyHandle],
	aclErr prim.Pointer[ACLError],
) FastlyStatus

// Lookup returns the entry for the IP, if it exists.
func (a *ACLHandle) Lookup(ip net.IP) (io.Reader, error) {
	body := HTTPBody{h: invalidBodyHandle}

	var ipBytes []byte
	if ipBytes = ip.To4(); ipBytes == nil {
		ipBytes = ip.To16()
	}
	ipBuffer := prim.NewReadBufferFromBytes(ipBytes).ArrayChar8()

	var aclErr ACLError = ACLErrorUninitialized

	if err := fastlyACLLookup(
		a.h,
		ipBuffer.Data, ipBuffer.Len,
		prim.ToPointer(&body.h),
		prim.ToPointer(&aclErr),
	).toError(); err != nil {
		return nil, err
	}

	if aclErr != ACLErrorOK {
		return nil, aclErr
	}

	// Didn't get a valid handle back.  This means there was no matching
	// ACL prefix.  Report back to caller we got no match.
	if body.h == invalidBodyHandle {
		return nil, ACLErrorNoContent
	}

	return &body, nil
}
