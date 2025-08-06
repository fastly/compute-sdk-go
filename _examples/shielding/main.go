// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/shielding"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		name := r.URL.Query().Get("shield")

		shield, err := shielding.ShieldFromName(name)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Shield Name=%v, RunningOn=%v\n", shield.Name(), shield.IsRunningOn())
	})
}
