//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2023 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
	"github.com/fastly/compute-sdk-go/geo"
)

func assert[T comparable](res fsthttp.ResponseWriter, field string, got, want T) {
	if got != want {
		fsthttp.Error(res, fmt.Sprintf("%s: got %v, want %v", field, got, want), fsthttp.StatusInternalServerError)
	}
}

func TestGeolocation(t *testing.T) {
	handler := func(ctx context.Context, res fsthttp.ResponseWriter, req *fsthttp.Request) {
		g, err := geo.Lookup(net.ParseIP("127.0.0.1"))
		if err != nil {
			fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		assert(res, "AsName", g.AsName, "Fastly Test")
		assert(res, "AsNumber", g.AsNumber, 12345)
		assert(res, "AreaCode", g.AreaCode, 123)
		assert(res, "City", g.City, "Test City")
		assert(res, "ConnSpeed", g.ConnSpeed, "broadband")
		assert(res, "ConnType", g.ConnType, "wired")
		assert(res, "ContinentCode", g.ContinentCode, "NA")
		assert(res, "CountryCode", g.CountryCode, "CA")
		assert(res, "CountryCode3", g.CountryCode3, "CAN")
		assert(res, "CountryName", g.CountryName, "Canada")
		assert(res, "Latitude", g.Latitude, 12.345)
		assert(res, "Longitude", g.Longitude, 54.321)
		assert(res, "MetroCode", g.MetroCode, 1)
		assert(res, "PostalCode", g.PostalCode, "12345")
		assert(res, "ProxyDescription", g.ProxyDescription, "?")
		assert(res, "ProxyType", g.ProxyType, "?")
		assert(res, "Region", g.Region, "BC")
		assert(res, "UTCOffset", g.UTCOffset, -700)
	}

	r, err := fsthttp.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := fsttest.NewRecorder()

	handler(context.Background(), w, r)

	if got, want := w.Code, fsthttp.StatusOK; got != want {
		t.Errorf("got %v, want %v", got, want)
		t.Error(w.Body.String())
	}
}
