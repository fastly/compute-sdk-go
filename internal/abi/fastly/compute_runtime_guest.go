//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls
// Copyright 2024 Fastly, Inc.
package fastly

import (
  "github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// witx:
// (module $fastly_compute_runtime
//   (@interface func (export "get_vcpu_ms")
//       (result $err (expected $vcpu_ms (error $fastly_status)))
//   )
// )
//
//go:wasmimport fastly_compute_runtime get_vcpu_ms
//go:noescape
func fastlyGetVCPUMs(prim.Pointer[prim.U64]) FastlyStatus

func GetVCPUMilliseconds() (uint64, error) {
  var milliseconds prim.U64

  err := fastlyGetVCPUMs(prim.ToPointer(&milliseconds)).toError()

  if err != nil {
    return 0, err
  }

  return uint64(milliseconds), nil
}
