// Copyright 2024 Fastly, Inc.

package main

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/fastly/compute-sdk-go/acl"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		aclh, err := acl.Open("example")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		ip := r.URL.Query().Get("ip")
		if ip == "" {
			ip = r.RemoteAddr
		}

		netip := net.ParseIP(ip)
		aclr, err := aclh.Lookup(netip)
		if errors.Is(err, acl.ErrNoContent) {
			fmt.Fprintln(w, "IP:", ip, "No Match")
			return
		}
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, "IP:", ip, "Prefix:", aclr.Prefix, "Action:", aclr.Action)
	})
}
