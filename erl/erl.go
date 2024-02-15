// Package erl provides Edge Rate Limiting functionality.
//
// This package includes a [RateCounter] type that can be used to
// increment an event counter, and to examine the rate of events per
// second within a POP over 1, 10, and 60 second windows.  It can also
// estimate the number of events seen over the past minute within a POP
// in 10 second buckets.
//
// The [PenaltyBox] type can be used to track entries that should be
// penalized for a certain amount of time.
//
// The [RateLimiter] type combines a rate counter and a penalty box to
// determine whether a given entry should be rate limited based on
// whether it exceeds a maximum threshold of events per second over a
// given rate window.  Most users can simply use [RateLimiter.CheckRate]
// rather than methods on [RateCounter] and [PenaltyBox].
//
// Rate counters and penalty boxes are combined and synchronized within
// a POP.  However, Edge Rate Limiting is not intended to compute counts
// or rates with high precision and may under count by up to 10%.
//
// Both rate counters and penalty boxes have a fixed capacity for
// entries.  Once a rate counter is full, each new entry evicts the
// entry that was least recently incremented.  Once a penalty box is
// full, each new entry will evict the entry with the shortest remaining
// time-to-live (TTL).
package erl

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

var (
	// ErrInvalidArgument indicates that an invalid argument was passed
	// to one of the edge rate limiter methods.
	//
	// Most functions and methods have limited ranges for their
	// parameters.  See the documentation for each call for more
	// details.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrUnexpected indicates that an unexpected error occurred.
	ErrUnexpected = errors.New("unexpected error")
)

var (
	// RateWindow1s incidates the rate of events per second over the past
	// second.
	RateWindow1s = fastly.RateWindow1s

	// RateWindow10s indicates the rate of events per second over the
	// past 10 seconds.
	RateWindow10s = fastly.RateWindow10s

	// RateWindow60s indicates the rate of events per second over the
	// past 60 seconds.
	RateWindow60s = fastly.RateWindow60s
)

var (
	// CounterDuration10s indicates the estimated number of events in
	// the most recent 10 second bucket.
	CounterDuration10s = fastly.CounterDuration10s

	// CounterDuration20s indicates the estimated number of events in
	// the most recent two 10 second buckets.
	CounterDuration20s = fastly.CounterDuration20s

	// CounterDuration30s indicates the estimated number of events in
	// the most recent three 10 second buckets.
	CounterDuration30s = fastly.CounterDuration30s
	// CounterDuration40s indicates the estimated number of events in
	// the most recent four 10 second buckets.
	CounterDuration40s = fastly.CounterDuration40s

	// CounterDuration50s indicates the estimated number of events in
	// the most recent five 10 second buckets.
	CounterDuration50s = fastly.CounterDuration50s

	// CounterDuration60s indicates the estimated number of events in
	// the most recent six 10 second buckets.
	CounterDuration60s = fastly.CounterDuration60s
)

type (
	// RateWindow indicates the rate of events per second in the current
	// POP over one of a few predefined time windows.  See
	// [RateWindow1s], [RateWindow10s], and [RateWindow60s].
	RateWindow = fastly.RateWindow

	// CounterDuration indicates the estimated number of events in this
	// duration in the current POP.  Counts are divided into 10 second
	// buckets, and each bucket represents the estimated number of
	// requests received up to and including that 10 second window.
	//
	// Buckets are not continuous.  For example, if the current time is
	// 12:01:03, then the 10 second bucket represents events received
	// between 12:01:00 and 12:01:10, not between 12:00:53 and 12:01:03.
	// This means that, in each minute at the ten second mark (:00, :10,
	// :20, etc.) the window represented by each bucket will shift.
	//
	// Estimated counts are not precise and should not be used as
	// counters.
	//
	// See [CounterDuration10s], [CounterDuration20s],
	// [CounterDuration30s], [CounterDuration40s], [CounterDuration50s],
	// and [CounterDuration60s].
	CounterDuration = fastly.CounterDuration
)

// RateCounter is a named counter that can be incremented and queried.
type RateCounter struct {
	name string
}

// OpenRateCounter opens a rate counter with the given name, creating it
// if it doesn't already exist.  The rate counter name may be up to 64
// characters long.  Entry names in this counter are also limited to 64
// characters.
func OpenRateCounter(name string) *RateCounter {
	return &RateCounter{name: name}
}

// Increment increments the rate counter for this entry by the given
// delta value.  The minimum value is 0 and the maximum is 1000.
func (rc *RateCounter) Increment(entry string, delta uint32) error {
	return mapFastlyError(fastly.RateCounterIncrement(rc.name, entry, delta))
}

// LookupRate returns the rate of events per second over the given rate
// window for this entry.
func (rc *RateCounter) LookupRate(entry string, window RateWindow) (uint32, error) {
	v, err := fastly.RateCounterLookupRate(rc.name, entry, window)
	if err != nil {
		return 0, mapFastlyError(err)
	}
	return v, nil
}

// LookupCount returns the estimated number of events in the given
// duration for this entry.  The duration represents a discrete window,
// not a continuous one.  See [CounterDuration] for more details.
func (rc *RateCounter) LookupCount(entry string, duration CounterDuration) (uint32, error) {
	v, err := fastly.RateCounterLookupCount(rc.name, entry, duration)
	if err != nil {
		return 0, mapFastlyError(err)
	}
	return v, nil
}

// PenaltyBox is a type that allows entries to penalized for a given
// number of minutes in the future.
type PenaltyBox struct {
	name string
}

// OpenPenaltyBox opens a penalty box with the given name, creating it
// if it doesn't already exist.  The penalty box name may be up to 64
// characters long.  Entry names in this penalty box are also limited to
// 64 characters.
func OpenPenaltyBox(name string) *PenaltyBox {
	return &PenaltyBox{name: name}
}

// Add adds an entry to the penalty box for the given time-to-live (TTL)
// duration.  The minimum value is 1 minute and the maximum is 60
// minutes.  If an entry is already in the penalty box, its TTL is
// replaced with the new value.  Entries are automatically evicted from
// the penalty box when the TTL expires.
func (pb *PenaltyBox) Add(entry string, ttl time.Duration) error {
	return mapFastlyError(fastly.PenaltyBoxAdd(pb.name, entry, ttl))
}

// Has returns true if the given entry is currently in the penalty box.
func (pb *PenaltyBox) Has(entry string) (bool, error) {
	ok, err := fastly.PenaltyBoxHas(pb.name, entry)
	if err != nil {
		return false, mapFastlyError(err)
	}
	return ok, nil
}

// Policy contains the rules for applying a [RateLimiter].
type Policy struct {
	// RateWindow is the window of time to consider when checking the
	// rate of events per second.
	RateWindow RateWindow

	// MaxRate is the maximum number of events per second to allow over
	// the rate window.  The minimum value is 10 and the maximum is
	// 10000.
	MaxRate uint32

	// PenaltyBoxDuration is the duration to penalize entries that
	// exceed the maximum rate.  As with PenaltyBox.Add, the minimum
	// value is 1 minute and the maximum is 60 minutes.
	PenaltyBoxDuration time.Duration
}

// UnmarshalJSON unmarshals a JSON value into a [Policy].  The JSON
// value must be in the form:
//
//	{
//	    "rate_window": 10,
//	    "max_rate": 100,
//	    "penalty_box_duration": 60
//	}
//
// The rate window must be one of the valid rate window values (1, 10,
// or 60), or an error is returned.  The penalty box duration is in
// minutes.
func (p *Policy) UnmarshalJSON(data []byte) error {
	var jsonPolicy struct {
		RateWindow         uint32 `json:"rate_window"` // Must be one of the valid rate window values.
		MaxRate            uint32 `json:"max_rate"`
		PenaltyBoxDuration uint32 `json:"penalty_box_duration"` // Value in minutes
	}
	if err := json.Unmarshal(data, &jsonPolicy); err != nil {
		return err
	}

	switch jsonPolicy.RateWindow {
	case 1:
		p.RateWindow = RateWindow1s
	case 10:
		p.RateWindow = RateWindow10s
	case 60:
		p.RateWindow = RateWindow60s
	default:
		return fmt.Errorf("invalid rate window: %d", jsonPolicy.RateWindow)
	}

	p.MaxRate = jsonPolicy.MaxRate
	p.PenaltyBoxDuration = time.Duration(jsonPolicy.PenaltyBoxDuration) * time.Minute
	return nil
}

// RateLimiter combines a [RateCounter] and a [PenaltyBox] to provide an
// easy way to check whether a given entry should be rate limited given
// a rate window and upper limit.
type RateLimiter struct {
	RateCounter *RateCounter
	PenaltyBox  *PenaltyBox
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(rc *RateCounter, pb *PenaltyBox) *RateLimiter {
	return &RateLimiter{
		RateCounter: rc,
		PenaltyBox:  pb,
	}
}

// CheckRate increments an entry's rate counter by the delta value
// ([RateCounter.Increment]), and checks it against the provided
// [Policy].  If the count after increment exceeds the policy's MaxRate
// over the RateWindow, it will add the entry to the penalty box for the
// policy's PenaltyBoxDuration.  It returns true if the entry is in the
// penalty box.
//
// The limits for the delta value are the same as
// [RateCounter.Increment].
func (erl *RateLimiter) CheckRate(entry string, delta uint32, policy *Policy) (bool, error) {
	blocked, err := fastly.ERLCheckRate(
		erl.RateCounter.name,
		entry,
		delta,
		policy.RateWindow,
		policy.MaxRate,
		erl.PenaltyBox.name,
		policy.PenaltyBoxDuration,
	)
	if err != nil {
		return false, mapFastlyError(err)
	}
	return blocked, nil
}

func mapFastlyError(err error) error {
	status, ok := fastly.IsFastlyError(err)
	if !ok {
		return err
	}

	switch status {
	case fastly.FastlyStatusInval:
		return ErrInvalidArgument
	default:
		return fmt.Errorf("%w (%s)", ErrUnexpected, status)
	}
}
