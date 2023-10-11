// Copyright 2022 Fastly, Inc.

// Package fastly provides access to the Fastly Compute hostcall ABI.
//
// The TinyGo SDK is modeled in layers. Each layer has a single purpose. This
// package is the lowest layer, and it's singular purpose is to adapt each
// Compute hostcall to a function which is basically idiomatic Go.
//
// In support of that purpose, the package defines a few types, e.g. HTTPBody,
// which model the modules of the hostcalls, and implement corresponding
// functions as methods on those types. Each hostcall should have a single
// corresponding Go method or function.
//
// There are also helper types, like Values, which make it easier to interact
// with the hostcall ABI. But, in general, this package should be kept as small
// as possible, and all nontrivial work performed at the next layer up, e.g.
// package fsthttp.
//
// This package is not and should not be user-accessible. All features,
// capabilities, etc. that should be accessible by users should be made
// available via separate packages that treat this package as a dependency.
package fastly
