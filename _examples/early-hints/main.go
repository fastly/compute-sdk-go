package main

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"net/http"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

//go:embed style.css
var styleCss []byte

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		if r.URL.Path == "/style.css" {
			w.Header().Set("Content-Type", "text/css")
			io.Copy(w, io.NopCloser(bytes.NewReader(styleCss)))
		} else {
			w.Header().Add("Link", "</style.css>; rel=preload; as=style")
			w.WriteHeader(http.StatusEarlyHints)
			resp, err := r.Send(ctx, "origin")
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
				return
			}
			w.Header().Reset(resp.Header)
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
		}
	})
}
