//go:build wasip1 && !nofastlyhostcalls

// Copyright 2024 Fastly, Inc.

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

// GetVCPUMilliseconds returns the number of milliseconds spent on the
// CPU for the current sandbox.
func GetVCPUMilliseconds() (uint64, error) {
	var milliseconds prim.U64

	if err := fastlyGetVCPUMs(prim.ToPointer(&milliseconds)).toError(); err != nil {
		return 0, err
	}

	return uint64(milliseconds), nil
}

// witx:
//
//	(module $fastly_compute_runtime
//	  (@interface func (export "get_heap_mib")
//	    (result $err (expected $memory_mib (error $fastly_status)))
//	  )
//	)
//
//go:wasmimport fastly_compute_runtime get_heap_mib
//go:noescape
func fastlyGetHeapMiB(prim.Pointer[prim.U32]) FastlyStatus

// GetHeapMiB returns the current memory usage of the sandbox in
// mebibytes.
func GetHeapMiB() (uint32, error) {
	var mib prim.U32

	if err := fastlyGetHeapMiB(prim.ToPointer(&mib)).toError(); err != nil {
		return 0, err
	}

	return uint32(mib), nil
}
