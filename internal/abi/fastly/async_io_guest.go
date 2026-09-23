//go:build wasip1 && !nofastlyhostcalls

// Copyright 2025 Fastly, Inc.

package fastly

import (
	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// asyncIOSelectTimedOut is the sentinel ready-index the host returns from
// `select` when the timeout elapses before any handle becomes ready.
const asyncIOSelectTimedOut = ^prim.U32(0)

// witx:
//
//	(@interface func (export "select")
//	    (param $hs (list $async_item_handle))
//	    (param $timeout_ms u32)
//	    (result $err (expected $ready_idx (error $fastly_status)))
//	)
//
//go:wasmimport fastly_async_io select
//go:noescape
func fastlyAsyncIOSelect(
	hs prim.Pointer[prim.U32], hsLen prim.Usize,
	timeoutMs prim.U32,
	readyIdx prim.Pointer[prim.U32],
) FastlyStatus

// HTTPCacheAwaitReady blocks for up to timeoutMs milliseconds waiting for the
// cache transaction lookup behind h to produce a result, and reports whether
// it became ready within that time. A timeoutMs of 0 blocks until h is
// ready.
func HTTPCacheAwaitReady(h *HTTPCacheHandle, timeoutMs uint32) (bool, error) {
	handle := prim.U32(h.h)
	var readyIdx prim.U32

	if err := fastlyAsyncIOSelect(
		prim.ToPointer(&handle), 1,
		prim.U32(timeoutMs),
		prim.ToPointer(&readyIdx),
	).toError(); err != nil {
		return false, err
	}

	return readyIdx != asyncIOSelectTimedOut, nil
}
