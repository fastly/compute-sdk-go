//go:build wasip1 && !nofastlyhostcalls

// Copyright 2023 Fastly, Inc.

package main

import (
	"testing"

	"github.com/fastly/compute-sdk-go/device"
)

func assert[T comparable](t *testing.T, field string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

func TestDeviceDetection(t *testing.T) {
	testcases := []struct {
		name      string
		userAgent string
	}{
		{
			name:      "iPhone",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64; rv:10.0) Gecko/20100101 Firefox/10.0 [FBAN/FBIOS;FBAV/8.0.0.28.18;FBBV/1665515;FBDV/iPhone4,1;FBMD/iPhone;FBSN/iPhone OS;FBSV/7.0.4;FBSS/2; FBCR/Telekom.de;FBID/phone;FBLC/de_DE;FBOP/5]",
		},

		{
			name:      "AsusTeK",
			userAgent: "ghosts-app/1.0.2.1 (ASUSTeK COMPUTER INC.; X550CC; Windows 8 (X86); en)",
		},

		{
			name:      "unknown",
			userAgent: "whoopty doopty doo",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := device.Lookup(tc.userAgent)

			switch tc.name {
			case "iPhone":
				if err != nil {
					t.Fatalf("Lookup: %v", err)
				}

				assert(t, "Name", d.Name(), "iPhone")
				assert(t, "Brand", d.Brand(), "Apple")
				assert(t, "Model", d.Model(), "iPhone4,1")
				assert(t, "HWType", d.HWType(), "Mobile Phone")
				assert(t, "IsMobile", d.IsMobile(), true)
				assert(t, "IsTouchscreen", d.IsTouchscreen(), true)

			case "AsusTeK":
				if err != nil {
					t.Fatalf("Lookup: %v", err)
				}

				assert(t, "Name", d.Name(), "Asus TeK")
				assert(t, "Brand", d.Brand(), "Asus")
				assert(t, "Model", d.Model(), "TeK")

			case "unknown":
				if err != device.ErrDeviceNotFound {
					t.Errorf("Lookup: got err %v, want %v", err, device.ErrDeviceNotFound)
				}
			}
		})
	}
}
