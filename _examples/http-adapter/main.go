package main

import (
	"io"
	"log"
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

	// Set a transport for the default HTTP client
	http.DefaultClient.Transport = fsthttp.NewTransport(backend)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	mux.HandleFunc("/ip", func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get("https://http-me.fastly.dev/ip")
		if err != nil {
			log.Println("error during fetch:", err)
			w.WriteHeader(http.StatusBadGateway)
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
