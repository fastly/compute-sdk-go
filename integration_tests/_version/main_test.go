// Copyright 2025 Fastly, Inc.

package main

import (
	"runtime/debug"
	"testing"
)

func TestGoVersion(t *testing.T) {
	bi, _ := debug.ReadBuildInfo()
	t.Log(bi.GoVersion)
}
