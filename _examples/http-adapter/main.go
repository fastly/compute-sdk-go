package main

import (
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

const backend = "httpme"

func main() {
	// http.ServeMux is an http.Handler implementation.  You can use any
	// one here, including chi, gorilla/mux, etc.
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	mux.HandleFunc("/ip", func(w http.ResponseWriter, r *http.Request) {
		req, err := fsthttp.NewRequest("GET", "https://http-me.fastly.dev/ip", nil)
		if err != nil {
			w.WriteHeader(fsthttp.StatusInternalServerError)
			return
		}

		req.Header.Set("Fastly-Debug", "1")

		resp, err := req.Send(r.Context(), backend)
		if err != nil {
			w.WriteHeader(fsthttp.StatusBadGateway)
			return
		}

		for k, v := range resp.Header {
			w.Header()[k] = v
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	mux.HandleFunc("/long", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		processTime := time.Duration(rand.Intn(10)+1) * time.Second

		select {
		case <-ctx.Done():
			return

		case <-time.After(processTime):
			// The above channel simulates some hard work.
		}

		w.Write([]byte("done"))
	})

	fsthttp.Serve(fsthttp.Adapt(mux))
}
