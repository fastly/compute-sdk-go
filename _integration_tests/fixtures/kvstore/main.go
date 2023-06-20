// Copyright 2023 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/kvstore"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, res fsthttp.ResponseWriter, req *fsthttp.Request) {
		store, err := kvstore.Open("example-test-kv-store")
		if err != nil {
			fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		switch req.URL.Path {
		case "/lookup":
			{
				hello, err := store.Lookup("hello")
				if err != nil {
					fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
					return
				}

				fmt.Fprint(res, hello.String())
			}
		case "/insert":
			{
				err := store.Insert("animal", strings.NewReader("cat"))
				if err != nil {
					fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
					return
				}

				animal, err := store.Lookup("animal")
				if err != nil {
					fsthttp.Error(res, err.Error(), fsthttp.StatusInternalServerError)
					return
				}

				fmt.Fprint(res, animal.String())
			}
		}

	})
}
