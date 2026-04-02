package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

// Response represents the complete response structure
type Response struct {
	ClientIP                   string `json:"client_ip"`
	*fsthttp.ResVPNProxyResult `json:",inline"`
}

// ErrorResponse represents an error response structure
type ErrorResponse struct {
	Error    string `json:"error"`
	ClientIP string `json:"client_ip"`
}

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Test the ResVPNProxy data using the request method
		vpnData, err := r.ResVPNProxyData()
		if err != nil {
			w.WriteHeader(fsthttp.StatusInternalServerError)
			errorResp := ErrorResponse{
				Error:    fmt.Sprintf("Error getting ResVPNProxy data: %s", err),
				ClientIP: r.RemoteAddr,
			}
			if jsonData, jsonErr := json.Marshal(errorResp); jsonErr != nil {
				fmt.Fprintf(w, `{"error": "JSON marshaling failed"}`)
			} else {
				w.Write(jsonData)
			}
			return
		}

		// Success case - return the ResVPNProxy data
		w.WriteHeader(fsthttp.StatusOK)
		response := Response{
			ClientIP:          r.RemoteAddr,
			ResVPNProxyResult: vpnData,
		}

		if jsonData, err := json.Marshal(response); err != nil {
			w.WriteHeader(fsthttp.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "JSON marshaling failed"}`)
		} else {
			w.Write(jsonData)
		}

		// Log to console for debugging
		log.Printf("ResVPNProxy analysis complete for client %s", r.RemoteAddr)
	})
}
