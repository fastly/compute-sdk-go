//go:build wasip1 && !nofastlyhostcalls

// Copyright 2022 Fastly, Inc.

package main

import (
	"fmt"
	"testing"

	"github.com/fastly/compute-sdk-go/shielding"
)

func TestShielding(t *testing.T) {

	var tests = []struct {
		shield, want string
	}{
		{"bfi-wa-us", "Name=bfi-wa-us RunningOn=false"},
		{"pdx-or-us", "Name=pdx-or-us RunningOn=true"},
	}

	for _, tt := range tests {
		shield, err := shielding.ShieldFromName(tt.shield)
		if err != nil {
			t.Errorf("ShieldFromName(%v) got error %v", tt.shield, err)
		}

		got := fmt.Sprintf("Name=%v RunningOn=%v", shield.Name(), shield.IsRunningOn())

		if got != tt.want {
			t.Errorf("Body = %q, want %q", got, tt.want)
		}
	}
}
