//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package fastly

import "github.com/fastly/compute-sdk-go/internal/abi/prim"

func init() {
	fastlyABIInit(1)
}

// witx:
//
//	(module $fastly_abi
//	 (@interface func (export "init")
//	   (param $abi_version u64)
//	   (result $err $fastly_status))
//	)
//
//go:wasmimport fastly_abi init
//go:noescape
func fastlyABIInit(abiVersion prim.U64) FastlyStatus

// TODO(pb): this doesn't need to be exported, I don't think?
// Initialize the Fastly ABI at the given version.
//func Initialize(version uint64) error {
