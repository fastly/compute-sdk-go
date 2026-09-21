package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

const GrpcOrigin = "grpc_origin"

func main() {
	// Log service version
	fmt.Println("FASTLY_SERVICE_VERSION:", os.Getenv("FASTLY_SERVICE_VERSION"))

	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		if !isGRPCRequest(r) {
			fsthttp.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		r.CacheOptions.Pass = true

		resp, err := r.Send(ctx, GrpcOrigin)
		if err != nil {
			log.Println("Error sending request to origin: ", err)
			fsthttp.Error(w, "Error sending request to origin", http.StatusInternalServerError)
			return
		}

		// copy the response headers
		w.Header().Reset(resp.Header)
		// Trailers are discovered after the response body has been read and are
		// forwarded below using TrailerPrefix.
		w.Header().Del("Trailer")
		w.WriteHeader(resp.StatusCode)

		// stream the response body to the client
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Println("Error copying response body: ", err)
			return
		}

		// copy the trailers to the response
		// first available when response body has been fully read
		trailers, err := resp.Trailers()
		if err != nil {
			log.Println("Error getting trailers: ", err)
			return
		}

		for _, key := range trailers.Keys() {
			w.Header().Set(
				fsthttp.TrailerPrefix+key,
				strings.Join(trailers.Values(key), ","),
			)
		}

		_ = w.Close()
	})
}

func isGRPCRequest(r *fsthttp.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}

	if mediaType == "application/grpc" {
		return true
	}

	suffix, ok := strings.CutPrefix(mediaType, "application/grpc+")
	if !ok {
		return false
	}

	return suffix != ""
}
