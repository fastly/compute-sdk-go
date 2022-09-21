// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"net"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/geo"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		ip := net.ParseIP(r.RemoteAddr)
		g, err := geo.Lookup(ip)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "AsName:           %q\n", g.AsName)
		fmt.Fprintf(w, "AsNumber:         %d\n", g.AsNumber)
		fmt.Fprintf(w, "AreaCode:         %d\n", g.AreaCode)
		fmt.Fprintf(w, "City:             %q\n", g.City)
		fmt.Fprintf(w, "ConnSpeed:        %q\n", g.ConnSpeed)
		fmt.Fprintf(w, "ConnType:         %q\n", g.ConnType)
		fmt.Fprintf(w, "ContinentCode:    %q\n", g.ContinentCode)
		fmt.Fprintf(w, "CountryCode:      %q\n", g.CountryCode)
		fmt.Fprintf(w, "CountryCode3:     %q\n", g.CountryCode3)
		fmt.Fprintf(w, "CountryName:      %q\n", g.CountryName)
		fmt.Fprintf(w, "Latitude:         %f\n", g.Latitude)
		fmt.Fprintf(w, "Longitude:        %f\n", g.Longitude)
		fmt.Fprintf(w, "MetroCode:        %d\n", g.MetroCode)
		fmt.Fprintf(w, "PostalCode:       %q\n", g.PostalCode)
		fmt.Fprintf(w, "ProxyDescription: %q\n", g.ProxyDescription)
		fmt.Fprintf(w, "ProxyType:        %q\n", g.ProxyType)
		fmt.Fprintf(w, "Region:           %q\n", g.Region)
		fmt.Fprintf(w, "UTCOffset:        %d\n", g.UTCOffset)
	})
}
