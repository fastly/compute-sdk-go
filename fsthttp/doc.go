// Copyright 2022 Fastly, Inc.

// Package fsthttp provides HTTP functionality for Fastly's Compute@Edge
// environment.
//
// A Compute@Edge program can be thought of as an HTTP request handler. Each
// execution is triggered by an incoming request from a client, and is expected
// to respond to that request before terminating. The Serve function provides a
// Handler-style interface to that Request and its ResponseWriter.
//
// The types in this package are similar to, but not the same as, corresponding
// types in the standard library's package net/http. Refer to the documentation
// for important caveats about usage.
package fsthttp
