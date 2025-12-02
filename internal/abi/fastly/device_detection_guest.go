//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import "github.com/fastly/compute-sdk-go/internal/abi/prim"

// witx:
//
//	(module $fastly_device_detection
//	    (@interface func (export "lookup")
//	        (param $user_agent string)
//
//	        (param $buf (@witx pointer (@witx char8)))
//	        (param $buf_len (@witx usize))
//	        (param $nwritten_out (@witx pointer (@witx usize)))
//	        (result $err (expected (error $fastly_status)))
//	    )
//	)
//
//go:wasmimport fastly_device_detection lookup
//go:noescape
func fastlyDeviceDetectionLookup(
	userAgentData prim.Pointer[prim.U8], userAgentLen prim.Usize,
	buf prim.Pointer[prim.Char8], bufLen prim.Usize,
	nWritten prim.Pointer[prim.Usize],
) FastlyStatus

func DeviceLookup(userAgent string) ([]byte, error) {
	userAgentBuffer := prim.NewReadBufferFromString(userAgent).Wstring()
	// Longest JSON of https://www.fastly.com/documentation/reference/vcl/variables/client-request/client-identified/
	value, err := withAdaptiveBuffer(DefaultMediumBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyDeviceDetectionLookup(
			userAgentBuffer.Data, userAgentBuffer.Len,
			prim.ToPointer(buf.Char8Pointer()),
			buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return nil, err
	}
	return value.AsBytes(), nil
}
