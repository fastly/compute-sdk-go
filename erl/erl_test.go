package erl_test

import (
	"context"
	"fmt"
	"time"

	"github.com/fastly/compute-sdk-go/erl"
	"github.com/fastly/compute-sdk-go/fsthttp"
)

func ExampleRateLimiter_CheckRate() {
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

func ExampleRateCounter_LookupRate() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		rc := erl.OpenRateCounter("requests")

		// Increment the request counter by 1
		rc.Increment(r.RemoteAddr, 1)

		// Get the current rate of requests per second over the past 60
		// seconds
		rate, err := rc.LookupRate(r.RemoteAddr, erl.RateWindow60s)
		if err != nil {
			w.WriteHeader(fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Rate over the past 60 seconds: %d requests per second\n", rate)
	})
}

func ExampleRateCounter_LookupCount() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		rc := erl.OpenRateCounter("requests")

		// Increment the request counter by 1
		rc.Increment(r.RemoteAddr, 1)

		// Get an estimated count of total number of requests over the
		// past 60 seconds
		count, err := rc.LookupCount(r.RemoteAddr, erl.CounterDuration60s)
		if err != nil {
			w.WriteHeader(fsthttp.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Estimated count over the past 60 seconds: %d requests\n", count)
	})
}
