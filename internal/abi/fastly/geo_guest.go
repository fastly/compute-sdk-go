//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import (
	"net"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
//
//	(module $fastly_geo
//	  (@interface func (export "lookup")
//	     (param $addr_octets (@witx pointer (@witx char8)))
//	     (param $addr_len (@witx usize))
//	     (param $buf (@witx pointer (@witx char8)))
//	     (param $buf_len (@witx usize))
//	     (param $nwritten_out (@witx pointer (@witx usize)))
//	     (result $err (expected (error $fastly_status)))
//	  )
//
// )
//
//go:wasmimport fastly_geo lookup
//go:noescape
func fastlyGeoLookup(
	addrOctets prim.Pointer[prim.Char8], addrLen prim.Usize,
	buf prim.Pointer[prim.Char8], bufLen prim.Usize,
	nWrittenOut prim.Pointer[prim.Usize],
) FastlyStatus

// GeoLookup returns the geographic data associated with the IP address.
func GeoLookup(ip net.IP) ([]byte, error) {
	if x := ip.To4(); x != nil {
		ip = x
	}
	addrOctets := prim.NewReadBufferFromBytes(ip)

	value, err := withAdaptiveBuffer(DefaultMediumBufLen, func(buf *prim.WriteBuffer) FastlyStatus {
		return fastlyGeoLookup(
			prim.ToPointer(addrOctets.Char8Pointer()), addrOctets.Len(),
			prim.ToPointer(buf.Char8Pointer()), buf.Cap(),
			prim.ToPointer(buf.NPointer()),
		)
	})
	if err != nil {
		return nil, err
	}
	return value.AsBytes(), nil
}
