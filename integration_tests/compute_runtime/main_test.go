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

  time.Sleep(5 * time.Second)

  end, err := compute.GetVCPUTime()
  if err != nil {
    t.Errorf("Couldn't get ending vcpu time")
  }

  if end - start > time.Second {
    t.Errorf("Sleeping shouldn't count as vcpu time!")
  }

  now, err := compute.GetVCPUTime()
  if err != nil {
    t.Errorf("Couldn't get starting vcpu time")
  }

  var counter uint64

  counter = 0
  next := now
  for now == next {
    new_next, err := compute.GetVCPUTime()
    if err != nil {
      t.Errorf("Couldn't get starting vcpu time")
    }
    next = new_next
    counter += 1
  }

  if counter == 0 {
    t.Errorf("It should take at least one loop to advance vcpu time")
  }
}
