// Copyright 2022 Fastly, Inc.

// Package fstctx provides alternatives to context.WithTimeout and
// context.WithDeadline. At the time of writing, TinyGo does not support the
// runtime methods needed to support those time based operations.
//
// All packages in `x`, including package fstctx, should be considered
// temporary, experimental, and unstable.
package fstctx
