// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		cookie := &fsthttp.Cookie{
			Name:     "Hello",
			Value:    r.RemoteAddr,
			Secure:   true,
			HttpOnly: true,
			SameSite: fsthttp.SameSiteStrictMode,
		}
		fsthttp.SetCookie(w.Header(), cookie)

		fmt.Fprintln(w, "Your cookie has been set!")
	})
}
