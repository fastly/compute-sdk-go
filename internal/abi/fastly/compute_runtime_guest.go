//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2024 Fastly, Inc.
//
package fastly

import (
	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
// (module $fastly_compute_runtime
//
//	(@interface func (export "get_vcpu_ms")
//	    (result $err (expected $vcpu_ms (error $fastly_status)))
//	)
//
// )
//
//go:wasmimport fastly_compute_runtime get_vcpu_ms
//go:noescape
func fastlyGetVCPUMs(prim.Pointer[prim.U64]) FastlyStatus

// Return the number of milliseconds spent on the CPU for the current
// session.
//
// Because compute guests can run on a variety of different platforms,
// you should not necessarily expect these values to converge across
// different sessions. Instead, we strongly recommend using this value
// to look at the relative cost of various operations in your code base,
// by taking the time before and after a particular operation and then
// dividing this by the total amount of vCPU time your program takes.
// The resulting percentage should be relatively stable across different
// platforms, and useful in doing A/B testing.
func GetVCPUMilliseconds() (uint64, error) {
	var milliseconds prim.U64

	err := fastlyGetVCPUMs(prim.ToPointer(&milliseconds)).toError()

	if err != nil {
		return 0, err
	}

	return uint64(milliseconds), nil
}
