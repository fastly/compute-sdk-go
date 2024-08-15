// Copyright 2024 Fastly, Inc.
package main

import (
  "testing"
  "time"

  "github.com/fastly/compute-sdk-go/compute_runtime"
)

func TestGetVcpuMs(t *testing.T) {
  start, err := compute_runtime.GetVCPUTime()
  if err != nil {
    t.Errorf("Couldn't get starting vcpu time")
  }

  time.Sleep(5 * time.Second)

  end, err := compute_runtime.GetVCPUTime()
  if err != nil {
    t.Errorf("Couldn't get ending vcpu time")
  }

  if end - start > time.Second {
    t.Errorf("Sleeping shouldn't count as vcpu time!")
  }
}
