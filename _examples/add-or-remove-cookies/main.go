// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/rtlog"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		endpoint := rtlog.Open("mylogs")

		// Read a specific cookie
		c, err := r.Cookie("myCookie")
		if err == nil {
			fmt.Fprintf(endpoint, "The value of myCookie is %s\n", c.Value)
		}

		// Remove all cookies from the request
		r.Header.Del("Cookie")

		resp, err := r.Send(ctx, "backend")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		// Set a cookie in the response
		fsthttp.SetCookie(resp.Header, &fsthttp.Cookie{
			Name:   "myCookie",
			Value:  "foo",
			Path:   "/",
			MaxAge: 60,
		})

		// You can set multiple cookies in one response
		resp.Header.Add("Set-Cookie", "mySecondCookie=bar; httpOnly")

		// It is usually a good idea to prevent downstream caching of
		// responses that set cookies
		resp.Header.Set("Cache-Control", "no-store, private")

		w.Header().Reset(resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
