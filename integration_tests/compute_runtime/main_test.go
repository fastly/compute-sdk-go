//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2024 Fastly, Inc.
package main

import (
  "testing"
  "time"

  "github.com/fastly/compute-sdk-go/compute"
)

func TestGetVcpuMs(t *testing.T) {
  start, err := compute.GetVCPUTime()
  if err != nil {
    t.Errorf("Couldn't get starting vcpu time")
  }

  time.Sleep(1 * time.Second)

  end, err := compute.GetVCPUTime()
  if err != nil {
    t.Errorf("Couldn't get ending vcpu time")
  }

  if end - start > (200 * time.Millisecond) {
    t.Errorf("Sleeping shouldn't count as vcpu time!")
  }

  now, err := compute.GetVCPUTime()
  if err != nil {
    t.Errorf("Couldn't get starting vcpu time (part 2)")
  }

  var counter uint64

  counter = 0
  next := now
  for now == next {
    new_next, err := compute.GetVCPUTime()
    if err != nil {
      t.Errorf("Couldn't get part 2's recheck of vcpu time")
    }
    next = new_next
    counter += 1
  }

  if counter == 0 {
    t.Errorf("It should take at least one loop to advance vcpu time")
  }
}
