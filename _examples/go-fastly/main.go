// Copyright 2026 Fastly, Inc.

// This example uses the go-fastly API client from a Compute service.
// Run locally with FASTLY_API_TOKEN set, then request /?id=<service ID>.
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/secretstore"
	"github.com/fastly/go-fastly/v19/fastly"
)

// backend must match a backend for https://api.fastly.com.
const backend = "fastly"

// serviceSummary is the JSON response body.
type serviceSummary struct {
	ServiceID     string     `json:"service_id"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	Comment       string     `json:"comment,omitempty"`
	CustomerID    string     `json:"customer_id"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	ActiveVersion *int       `json:"active_version,omitempty"`
	LatestVersion int        `json:"latest_version"`
	VersionCount  int        `json:"version_count"`
}

func main() {
	token, err := secretstore.Plaintext("tokens", "fastly")
	if err != nil {
		log.Fatal("unable to open secret store: ", err)
	}

	client, err := fastly.NewClient(string(token))
	if err != nil {
		log.Fatal("error: NewClient: ", err)
	}

	// Compute has no raw networking, so send requests to the backend.
	client.HTTPClient.Transport = fsthttp.NewTransport(backend)

	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			fsthttp.Error(w, "missing id", fsthttp.StatusBadRequest)
			return
		}

		svc, err := client.GetServiceDetails(ctx, &fastly.GetServiceDetailsInput{ServiceID: id})
		if err != nil {
			log.Println("GetServiceDetails:", err)
			fsthttp.Error(w, "internal server error", fsthttp.StatusInternalServerError)
			return
		}

		summary := serviceSummary{
			ServiceID:    fastly.ToValue(svc.ServiceID),
			Name:         fastly.ToValue(svc.Name),
			Type:         fastly.ToValue(svc.Type),
			Comment:      fastly.ToValue(svc.Comment),
			CustomerID:   fastly.ToValue(svc.CustomerID),
			CreatedAt:    svc.CreatedAt,
			UpdatedAt:    svc.UpdatedAt,
			VersionCount: len(svc.Versions),
		}

		// ActiveVersion is nil for a service that was never activated.
		if svc.ActiveVersion != nil {
			summary.ActiveVersion = svc.ActiveVersion.Number
		}
		if svc.Version != nil {
			summary.LatestVersion = fastly.ToValue(svc.Version.Number)
		}

		body, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			log.Println("json.MarshalIndent:", err)
			fsthttp.Error(w, "internal server error", fsthttp.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(append(body, '\n'))
	})
}
