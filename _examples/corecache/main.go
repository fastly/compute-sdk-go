package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fastly/compute-sdk-go/cache/core"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		key := keyForRequest(r)
		switch r.Method {

		// Fetch content from the cache.
		case "GET":
			tx, err := core.NewTransaction(key, core.LookupOptions{})
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}
			defer tx.Close()

			f, err := tx.Found()
			if errors.Is(err, core.ErrNotFound) {
				fsthttp.Error(w, err.Error(), fsthttp.StatusNotFound)
				return
			}
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}
			defer f.Body.Close()

			msg, err := io.ReadAll(f.Body)
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "%s's message for %s is: %s\n", getPOP(), r.URL.Path, msg)

		// Write data to the cache and stream it back to the client.
		case "POST":
			if r.Header.Get("Content-Type") != "text/plain" && r.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
				w.WriteHeader(fsthttp.StatusUnsupportedMediaType)
				return
			}

			msg, err := io.ReadAll(r.Body)
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

			tx, err := core.NewTransaction(key, core.LookupOptions{})
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}
			defer tx.Close()

			if !tx.MustInsert() {
				w.WriteHeader(fsthttp.StatusConflict)
				return
			}

			// We call InsertAndStreamBack to create both a handle to
			// write the content into the cache and a Found object to
			// stream the contents back out to the client.  As soon as
			// data is written to the insert body, it is immediately
			// streamed to all clients waiting on this transaction,
			// including this one.
			//
			// This is preferable to using an io.MultiWriter because a
			// MultiWriter is constrained by the slowest writer.  If
			// other transactions are waiting on the content to be
			// written to the cache and streamed, we don't want that
			// process to be delayed by a slow client for this request.
			insertBody, found, err := tx.InsertAndStreamBack(core.WriteOptions{
				TTL:           600 * time.Second,
				Length:        uint64(len(msg)),
				SurrogateKeys: []string{hex.EncodeToString(key)},
			})
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}
			defer found.Body.Close()

			insertBody.Write(msg)

			if err := insertBody.Close(); err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

			msg, err = io.ReadAll(found.Body)
			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(fsthttp.StatusCreated)
			fmt.Fprintf(w, "%s's message for %s is: %s\n", getPOP(), r.URL.Path, msg)

		// Purge the key from the cache.
		case "DELETE":
			// TODO: purge the surrogate key.
			w.WriteHeader(fsthttp.StatusNotImplemented)

		default:
			w.WriteHeader(fsthttp.StatusMethodNotAllowed)
		}
	})
}

func keyForRequest(r *fsthttp.Request) []byte {
	h := sha256.New()
	h.Write([]byte(r.URL.Path))
	h.Write([]byte(getPOP()))
	return h.Sum(nil)
}

func getPOP() string {
	return os.Getenv("FASTLY_POP")
}
