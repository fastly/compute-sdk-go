package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fastly/compute-sdk-go/cache/simple"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		w.Header().Set("Service-Version", os.Getenv("FASTLY_SERVICE_VERSION"))

		key := keyForRequest(r)
		switch r.Method {

		// Fetch content from the cache.
		case "GET":
			rc, err := simple.Get(key)
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}
			defer rc.Close()

			msg, err := io.ReadAll(rc)
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "%s's message for %s is: %s\n", getPOP(), r.URL.Path, msg)

		// Write data to the cache (if there's nothing there) and stream it back to the client.
		case "POST":
			if r.Header.Get("Content-Type") != "text/plain" && r.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
				w.WriteHeader(fsthttp.StatusUnsupportedMediaType)
				return
			}

			var set bool
			rc, err := simple.GetOrSet(key, func() (simple.CacheEntry, error) {
				set = true
				return simple.CacheEntry{
					Body: r.Body,
					TTL:  3 * time.Minute,
				}, nil
			})
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}
			defer rc.Close()

			if !set {
				w.WriteHeader(fsthttp.StatusConflict)
				return
			}

			msg, err := io.ReadAll(rc)
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(fsthttp.StatusOK)
			fmt.Fprintf(w, "%s's message for %s is: %s\n", getPOP(), r.URL.Path, msg)

		// Purge the key from the cache.
		case "DELETE":
			if err := simple.Purge(key, simple.PurgeOptions{}); err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}
			w.WriteHeader(fsthttp.StatusAccepted)

		default:
			w.WriteHeader(fsthttp.StatusMethodNotAllowed)
		}
	})
}

func keyForRequest(r *fsthttp.Request) []byte {
	h := sha256.New()
	h.Write([]byte(r.URL.Path))
	return h.Sum(nil)
}

func getPOP() string {
	return os.Getenv("FASTLY_POP")
}
