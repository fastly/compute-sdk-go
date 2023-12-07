package erl_test

import (
	"context"
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
			r.RemoteAddr,      // Use the IP address of the client as the entry
			1,                 // Increment the request counter by 1
			erl.RateWindow10s, // Check the rate of requests per second over the past 10 seconds
			100,               // Allow up to 100 requests per second
			time.Minute,       // Put offenders into the penalty box for 1 minute
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
