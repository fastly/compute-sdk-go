//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

// Copyright 2023 Fastly, Inc.

package main

import (
	"net"
	"testing"

	"github.com/fastly/compute-sdk-go/geo"
)

func assert[T comparable](t *testing.T, field string, got, want T) {
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

func TestGeolocation(t *testing.T) {
	g, err := geo.Lookup(net.ParseIP("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}

	assert(t, "AsName", g.AsName, "Fastly Test")
	assert(t, "AsNumber", g.AsNumber, 12345)
	assert(t, "AreaCode", g.AreaCode, 123)
	assert(t, "City", g.City, "Test City")
	assert(t, "ConnSpeed", g.ConnSpeed, "broadband")
	assert(t, "ConnType", g.ConnType, "wired")
	assert(t, "ContinentCode", g.ContinentCode, "NA")
	assert(t, "CountryCode", g.CountryCode, "CA")
	assert(t, "CountryCode3", g.CountryCode3, "CAN")
	assert(t, "CountryName", g.CountryName, "Canada")
	assert(t, "Latitude", g.Latitude, 12.345)
	assert(t, "Longitude", g.Longitude, 54.321)
	assert(t, "MetroCode", g.MetroCode, 1)
	assert(t, "PostalCode", g.PostalCode, "12345")
	assert(t, "ProxyDescription", g.ProxyDescription, "?")
	assert(t, "ProxyType", g.ProxyType, "?")
	assert(t, "Region", g.Region, "BC")
	assert(t, "UTCOffset", g.UTCOffset, -700)
}

func BenchmarkGeo(b *testing.B) {
	ip := net.ParseIP("127.0.0.1")
	var (
		g   *geo.Geo
		err error
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, err = geo.Lookup(ip)
		if err != nil {
			b.Fatal(err)
		}
	}
	if g.AsName != "Fastly Test" {
		b.Fatalf("AsName: got %v, want %v", g.AsName, "Fastly Test")
	}
}
