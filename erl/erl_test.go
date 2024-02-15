package erl_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/fastly/compute-sdk-go/erl"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

func ExampleRateLimiter() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		limiter := erl.NewRateLimiter(
			erl.OpenRateCounter("requests"),
			erl.OpenPenaltyBox("bad_ips"),
		)

		block, err := limiter.CheckRate(
			r.RemoteAddr, // Use the IP address of the client as the entry
			1,            // Increment the request counter by 1
			&erl.Policy{
				erl.RateWindow10s, // Check the rate of requests per second over the past 10 seconds
				100,               // Allow up to 100 requests per second
				time.Minute,       // Put offenders into the penalty box for 1 minute
			},
		)
		if err != nil {
			// It's probably better to fail open.  Consider logging the
			// error but continuing to handle the request.
		} else if block {
			// The rate limit has been exceeded.  Return a 429 Too Many
			// Requests response.
			w.WriteHeader(fsthttp.StatusTooManyRequests)
			return
		}

		// Otherwise, continue processing the request.
	})
}

func TestPolicyUnmarshalJSON(t *testing.T) {
	testcases := []struct {
		name    string
		data    string
		wantVal *erl.Policy
		wantErr error
	}{
		{
			name: "valid",
			data: `{"rate_window":10,"max_rate":100,"penalty_box_duration":60}`,
			wantVal: &erl.Policy{
				RateWindow:         erl.RateWindow10s,
				MaxRate:            100,
				PenaltyBoxDuration: time.Hour,
			},
		},

		{
			name:    "invalid rate window",
			data:    `{"rate_window":5,"max_rate":100,"penalty_box_duration":60}`,
			wantErr: fmt.Errorf("invalid rate window: 5"),
		},

		{
			name:    "missing rate window",
			data:    `{"max_rate":100,"penalty_box_duration":60}`,
			wantErr: fmt.Errorf("invalid rate window: 0"),
		},

		{
			name:    "invalid JSON",
			data:    `{"rate_window":10,"max_rate":100,"penalty_box_duration":60`,
			wantErr: fmt.Errorf("unexpected end of JSON input"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var p erl.Policy
			err := json.Unmarshal([]byte(tc.data), &p)

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("got nil, want %v", tc.wantErr)
				}
				if err.Error() != tc.wantErr.Error() {
					t.Errorf("got %v, want %v", err, tc.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("got %v, want nil", err)
				}

				if !reflect.DeepEqual(&p, tc.wantVal) {
					t.Errorf("got %v, want %v", &p, tc.wantVal)
				}
			}
		})
	}
}
