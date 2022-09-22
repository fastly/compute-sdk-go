// Copyright 2022 Fastly, Inc.

package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"strconv"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func isVowel(b byte) bool {
	switch b {
	case 'a', 'A', 'e', 'E', 'i', 'I', 'o', 'O', 'u', 'U':
		return true
	}
	return false
}

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		req, err := fsthttp.NewRequest("GET", "https://compute-sdk-test-backend.edgecompute.app/streaming_close", nil)
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			return
		}

		req.CacheOptions.Pass = true
		resp, err := req.Send(ctx, "TheOrigin")
		if err != nil {
			fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
			return
		}

		var removed int

		br := bufio.NewReader(resp.Body)
		for {
			b, err := br.ReadByte()
			if err == io.EOF {
				break
			}

			if err != nil {
				fsthttp.Error(w, err.Error(), fsthttp.StatusBadGateway)
				return
			}

			if isVowel(b) {
				removed++
			} else {
				w.Write([]byte{b})
			}
		}

		w.Close()

		treq, err := fsthttp.NewRequest("POST", "http://telemetry-server.com/example", nil)
		if err != nil {
			log.Printf("Constructing telemetry request: %v", err)
			return
		}

		treq.Header.Set("Vowels-Removed", strconv.Itoa(removed))
		if _, err = treq.Send(ctx, "TelemetryServer"); err != nil {
			log.Printf("Sending telemetry data: %v", err)
			return
		}
	})
}
