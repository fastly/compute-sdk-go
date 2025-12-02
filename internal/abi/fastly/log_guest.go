//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// (module $fastly_log

// LogEndpoint represents a specific Fastly log endpoint.
type LogEndpoint struct {
	h endpointHandle
}

// witx:
//
//	(@interface func (export "endpoint_get")
//	  (param $name (array u8))
//	  (result $err $fastly_status)
//	  (result $endpoint_handle_out $endpoint_handle))
//
//go:wasmimport fastly_log endpoint_get
//go:noescape
func fastlyLogEndpointGet(
	nameData prim.Pointer[prim.U8], nameLen prim.Usize,
	endpointHandleOut prim.Pointer[endpointHandle],
) FastlyStatus

// GetLogEndpoint opens the log endpoint identified by name.
func GetLogEndpoint(name string) (*LogEndpoint, error) {
	var e LogEndpoint

	nameBuffer := prim.NewReadBufferFromString(name).ArrayU8()

	if err := fastlyLogEndpointGet(
		nameBuffer.Data, nameBuffer.Len,
		prim.ToPointer(&e.h),
	).toError(); err != nil {
		return nil, err
	}

	return &e, nil
}

// witx:
//
//	(@interface func (export "write")
//	  (param $h $endpoint_handle)
//	  (param $msg (array u8))
//	  (result $err $fastly_status)
//	  (result $nwritten_out (@witx usize)))
//
// )
//
//go:wasmimport fastly_log write
//go:noescape
func fastlyLogWrite(
	h endpointHandle,
	msgData prim.Pointer[prim.U8], msgLen prim.Usize,
	nWrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// Write implements io.Writer, writing len(p) bytes from p into the endpoint.
// Returns the number of bytes written, and any error encountered.
// By contract, if n < len(p), the returned error will be non-nil.
func (e *LogEndpoint) Write(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		var nWritten prim.Usize
		p_n_Buffer := prim.NewReadBufferFromBytes(p[n:]).ArrayU8()

		if err = fastlyLogWrite(
			e.h,
			p_n_Buffer.Data, p_n_Buffer.Len,
			prim.ToPointer(&nWritten),
		).toError(); err == nil {
			n += int(nWritten)
		}
	}
	return n, err
}
