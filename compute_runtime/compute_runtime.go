// Copyright 2024 Fastly, Inc.
//
package compute_runtime

import (
  "time"

  "github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

func GetVCPUTime() (time.Duration, error) {
  milliseconds, err := fastly.GetVCPUMilliseconds()

  if err != nil {
    return 0, err
  }

  var result time.Duration
  result = time.Duration(milliseconds) * time.Millisecond 

  return result, nil
}
