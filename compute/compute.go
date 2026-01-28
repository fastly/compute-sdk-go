// Copyright 2024 Fastly, Inc.

// Useful functions for interacting with the compute instance runtime.
package compute

import (
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

// Get the amount of time taken on the vCPU.
//
// The resulting time is millisecond-accurate, but we recommend against
// comparing the absolute values returned across different runs (or builds)
// of the program.
//
// Because compute guests can run on a variety of different platforms,
// you should not necessarily expect these raw values to converge across
// different executions. Instead, we strongly recommend using this value
// to look at the relative cost of various operations in your code base,
// by taking the time before and after a particular operation and then
// dividing this by the total amount of vCPU time your program takes.
// The resulting percentage should be relatively stable across different
// platforms, and useful in doing A/B testing.
func GetVCPUTime() (time.Duration, error) {
	milliseconds, err := fastly.GetVCPUMilliseconds()
	if err != nil {
		return 0, err
	}

	result := time.Duration(milliseconds) * time.Millisecond

	return result, nil
}

// Get the current dynamic memory usage of this sandbox, rounded up to
// the nearest mebibyte (2^20 bytes).
//
// The memory accounted includes the Wasm linear memory (i.e. heap
// memory), as well as some memory used by the hosting environment; for
// instance, HTTP bodies that have been read from a TCP connection, but
// not read by the Wasm program. As such, this function provides only a
// snapshot in time: the memory usage can change without any action from
// the Wasm program. It can also change across runs, as the Compute
// platform's memory usage changes. Consider the returned value with
// these possibilities in mind.
func GetHeapMiB() (uint32, error) {
	return fastly.GetHeapMiB()
}
