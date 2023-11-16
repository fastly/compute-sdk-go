//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2023 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/fastly/compute-sdk-go/device"
	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func assert[T comparable](res fsthttp.ResponseWriter, field string, got, want T) {
	if got != want {
		fsthttp.Error(res, fmt.Sprintf("%s: got %v, want %v", field, got, want), fsthttp.StatusInternalServerError)
	}
}

func TestDeviceDetection(t *testing.T) {
	handler := func(ctx context.Context, res fsthttp.ResponseWriter, req *fsthttp.Request) {
		d, err := device.Lookup(req.Header.Get("User-Agent"))

		switch req.URL.Path {
		case "/iPhone":
			if err != nil {
				fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

			assert(res, "Name", d.Name(), "iPhone")
			assert(res, "Brand", d.Brand(), "Apple")
			assert(res, "Model", d.Model(), "iPhone4,1")
			assert(res, "HWType", d.HWType(), "Mobile Phone")
			assert(res, "IsMobile", d.IsMobile(), true)
			assert(res, "IsTouchscreen", d.IsTouchscreen(), true)

		case "/AsusTeK":
			if err != nil {
				fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

			assert(res, "Name", d.Name(), "Asus TeK")
			assert(res, "Brand", d.Brand(), "Asus")
			assert(res, "Model", d.Model(), "TeK")

		case "/unknown":
			if err != device.ErrDeviceNotFound {
				fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

		default:
			fsthttp.Error(res, "not found", fsthttp.StatusNotFound)
		}
	}

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
			r, err := fsthttp.NewRequest("GET", "/"+tc.name, nil)
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("User-Agent", tc.userAgent)
			w := fsttest.NewRecorder()

			handler(context.Background(), w, r)

			if got, want := w.Code, fsthttp.StatusOK; got != want {
				t.Errorf("got %v, want %v", got, want)
				t.Error(w.Body.String())
			}
		})
	}
}
