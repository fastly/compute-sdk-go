package main

import (
	"fmt"
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
		req, err := fsthttp.NewRequest("GET", "https://http-me.glitch.me/ip", nil)
		if err != nil {
			w.WriteHeader(fsthttp.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		req.Header.Set("Fastly-Debug", "1")

		resp, err := req.Send(r.Context(), backend)
		if err != nil {
			w.WriteHeader(fsthttp.StatusBadGateway)
			w.Write([]byte(err.Error()))
			return
		}

		for k, v := range resp.Header {
			w.Header()[k] = v
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		fmt.Fprintf(w, "\n---\n")

		ofr := fsthttp.RequestFromContext(r.Context())
		fmt.Fprintf(w, "%s\n", ofr.Host)
	})

	mux.HandleFunc("/long", func(w http.ResponseWriter, r *http.Request) {
		rand.Seed(time.Now().UnixNano())

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
