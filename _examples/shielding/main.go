// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/shielding"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		name := r.URL.Query().Get("shield")

		shield, err := shielding.ShieldFromName(name)
		if err != nil {
			log.Println("error looking up shield:", err)
			fsthttp.Error(w, fsthttp.StatusText(fsthttp.StatusInternalServerError), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Shield Name=%v, RunningOn=%v\n", shield.Name(), shield.IsRunningOn())
	})
}
