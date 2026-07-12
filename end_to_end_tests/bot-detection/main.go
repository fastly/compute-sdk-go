//go:build !test

package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {

	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		b, err := r.BotDetection()
		if err != nil {
			log.Println("error during bot detection:", err)
			fsthttp.Error(w, fsthttp.StatusText(fsthttp.StatusInternalServerError), fsthttp.StatusInternalServerError)
			return
		}

		jb, err := json.Marshal(b)
		if err != nil {
			log.Println("error during json marshal:", err)
			fsthttp.Error(w, fsthttp.StatusText(fsthttp.StatusInternalServerError), fsthttp.StatusInternalServerError)
			return
		}
		w.Write(jb)
	})
}
