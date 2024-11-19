//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2023 Fastly, Inc.

package main

import (
	"errors"
	"net"
	"testing"

	"github.com/fastly/compute-sdk-go/acl"
)

func TestACL(t *testing.T) {
	store, err := acl.Open("example")
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		ip  string
		r   acl.Response
		err error
	}{
		{"1.2.3.4", acl.Response{Prefix: "1.2.3.4/32", Action: "ALLOW"}, nil},
		{"1.1.1.1", acl.Response{}, acl.ErrNoContent},
		{"1.1.1", acl.Response{}, acl.ErrInvalidArgument},
	}

	for _, tt := range tests {

		lookup, err := store.Lookup(net.ParseIP(tt.ip))
		if (tt.err == nil && err != nil) || (tt.err != nil && !errors.Is(err, tt.err)) {
			t.Errorf("Lookup(%v) error mismatch: got %v, want %v", tt.ip, err, tt.err)
			continue
		}

		if lookup.Prefix != tt.r.Prefix || lookup.Action != tt.r.Action {
			t.Errorf("Lookup(%v) mismatch: got %#v, want %#v\n", tt.ip, lookup, tt.r)
		}

	}

	store, err = acl.Open("does-not-exist")
	if err != acl.ErrNotFound {
		t.Errorf("Open(does-not-exist) err = %v, want %v\n", err, acl.ErrNotFound)
	}
}
