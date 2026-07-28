//go:build wasip1 && !nofastlyhostcalls

package main

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func addViceroyBotDetectionHeaders(r *fsthttp.Request, b *fsthttp.BotDetectionResult) {
	if b.Analyzed {
		r.Header.Add("X-Fastly-Bot-Analyzed", "true")
	}

	if b.Category != fsthttp.BotCategoryNone {
		r.Header.Add("X-Fastly-Bot-Category", strconv.Itoa(int(b.Category)))
	}

	if b.Detected {
		r.Header.Add("X-Fastly-Bot-Detected", "true")
	}

	if b.Name != "" {
		r.Header.Add("X-Fastly-Bot-Name", b.Name)
	}

	if b.Verified {
		r.Header.Add("X-Fastly-Bot-Verified", "true")
	}
}

func TestBotDetection(t *testing.T) {

	tests := []fsthttp.BotDetectionResult{
		{},

		{
			Analyzed: true,
			Detected: false,
		},
		{
			Analyzed: true,
			Detected: true,
			Verified: true,
			Category: fsthttp.BotCategorySearchEngineCrawler,
			Name:     "crawler-bot",
		},

		{
			Analyzed: true,
			Detected: true,
			Verified: true,
			Category: fsthttp.BotCategoryHeadless,
			Name:     "ichabot",
		},
	}

	for _, want := range tests {
		req, err := fsthttp.NewRequest("GET", "http://anyplace.horse", nil)
		if err != nil {
			t.Fatal(err)
		}
		addViceroyBotDetectionHeaders(req, &want)

		resp, err := req.Send(context.Background(), "self")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fsthttp.StatusOK {
			t.Fatalf("unexpected status: got %d, want %d", resp.StatusCode, fsthttp.StatusOK)
		}

		d := json.NewDecoder(resp.Body)

		var got fsthttp.BotDetectionResult
		if err := d.Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		// CategoryName strings can vary
		got.CategoryName = ""

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("mismatch: got %#v, want %#v", got, want)
		}
	}
}
