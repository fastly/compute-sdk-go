// Copyright 2022 Fastly, Inc.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

var ErrSkip = errors.New("skipped")

func main() {

	println("service built with", runtime.Compiler)

	var tests = []struct {
		name string
		t    func(context.Context) error
	}{
		{"testSendStatus", testSendStatus},
		{"testBeforeSendError", testBeforeSendError},
		{"testBeforeSendPass", testBeforeSendPass},
		{"testAfterSendPass", testAfterSendPass},
		{"testBeforeSendAddHeader", testBeforeSendAddHeader},
		{"testAfterSendError", testAfterSendError},
		{"testAfterSendCandidateResponsePropertiesUncached", testAfterSendCandidateResponsePropertiesUncached},
		{"testAfterSendCandidateResponsePropertiesCached", testAfterSendCandidateResponsePropertiesCached},
		{"testAfterSendNoCache", testAfterSendNoCache},
		{"testAfterSendCached", testAfterSendCached},
		{"testAfterSendExpire", testAfterSendExpire},
		{"testAfterSendHeaderRemove", testAfterSendHeaderRemove},
		{"testAfterSendBodyTransform", testAfterSendBodyTransform},
		{"testAfterSendBodyTransformFailedCacheable", testAfterSendBodyTransformFailedCacheable},
		{"testAfterSendBodyTransformFailedUncacheable", testAfterSendBodyTransformFailedUncacheable},
		{"testAfterSendMutations", testAfterSendMutations},
		{"testRequestCollapse", testRequestCollapse},
		{"testRequestCollapseThree", testRequestCollapseThree},
		{"testRequestCollapseUncacheable", testRequestCollapseUncacheable},
		{"testRequestCollapseVary", testRequestCollapseVary},
		{"testError", testError},
	}

	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		test := r.URL.Path[1:]
		var run int

		if test == "all" {
			// don't run testError for `all`
			tests = tests[:len(tests)-1]
			rand.Shuffle(len(tests), func(i, j int) { tests[i], tests[j] = tests[j], tests[i] })
		}

		for _, tt := range tests {
			if test == "all" || tt.name == test {
				fmt.Fprintf(w, "=== %v: ", tt.name)
				run++
				switch err := tt.t(ctx); err {
				case ErrSkip:
					fmt.Fprintln(w, "SKIPPED")
				case nil:
					fmt.Fprintln(w, "PASS")
				default:
					fmt.Fprintf(w, "FAILED:\n---\n%v\n---\n", err)
				}
			}
		}
		if run == 0 {
			fsthttp.NotFound(w, r)
		}
	})
}

type query struct {
	status *int
	wait   time.Duration
}

func (q *query) String() string {
	if q == nil {
		return ""
	}

	var args []string

	if q.status != nil {
		s := fmt.Sprintf("status=%v", *q.status)
		args = append(args, s)
	}

	if q.wait != 0 {
		s := fmt.Sprintf("wait=%v", q.wait.Milliseconds())
		args = append(args, s)
	}

	return "?" + strings.Join(args, "&")
}

func getTestReq(method string, q *query, body io.Reader) *fsthttp.Request {
	if method == "" {
		method = "GET"
	}

	uri := "https://http-me.fastly.dev/anything/" + strconv.Itoa(rand.Int()) + "/" + q.String()
	println("uri=", uri)
	req, err := fsthttp.NewRequest(method, uri, body)
	if err != nil {
		println("error during NewRequest: uri=", uri, "err=", err)
	}

	return req
}

const backend = "httpme"

type anythingJSON struct {
	Args    map[string]string `json:"args"`
	Headers map[string]string `json:"headers"`
	Method  string            `json:"method"`
	Origin  string            `json:"origin"`
	URL     string            `json:"url"`
}

func decodeBody(r io.Reader) *anythingJSON {
	var j anythingJSON
	dec := json.NewDecoder(r)
	dec.Decode(&j)
	return &j
}

func testError(context.Context) error {
	return errors.New("test error")
}

func testSendStatus(ctx context.Context) error {

	r := getTestReq("", nil, nil)
	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if got, want := resp.StatusCode, 200; got != want {
		return fmt.Errorf("testSendStatus Ok: bad status code: got=%v, want=%v", got, want)
	}

	teapot := fsthttp.StatusTeapot

	r = getTestReq("", &query{status: &teapot}, nil)
	resp, err = r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if got, want := resp.StatusCode, teapot; got != want {
		return fmt.Errorf("testSendStatus teapot: bad status code: got=%v, want=%v", got, want)
	}

	return nil
}

func testBeforeSendError(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var errBeforeSend = errors.New("before send error")

	r.CacheOptions.BeforeSend = func(r *fsthttp.Request) error {
		return errBeforeSend
	}

	_, err := r.Send(ctx, backend)
	if !errors.Is(err, errBeforeSend) {
		return fmt.Errorf("unexpected error: got %v, want %v", err, errBeforeSend)
	}

	return nil
}

func testBeforeSendPass(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var calledBeforeSend bool

	r.CacheOptions.Pass = true
	r.CacheOptions.BeforeSend = func(r *fsthttp.Request) error {
		calledBeforeSend = true
		return nil
	}

	_, err := r.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("unexpected Send error: %v", err)
	}
	if calledBeforeSend {
		return fmt.Errorf("unexpected before send called, want false")
	}

	return nil
}

func testBeforeSendAddHeader(ctx context.Context) error {

	r := getTestReq("", nil, nil)

	var calledBeforeSend bool

	const (
		headerKey = "x-test"
		headerVal = "modified value"
	)

	r.CacheOptions.BeforeSend = func(r *fsthttp.Request) error {
		calledBeforeSend = true
		r.Header.Set(headerKey, headerVal)
		return nil
	}

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledBeforeSend {
		return errors.New("calledBeforeSend=false")
	}

	j := decodeBody(resp.Body)

	if got, want := j.Headers[headerKey], headerVal; got != want {
		return fmt.Errorf("bad added test header: got=%v, want=%v", got, want)
	}

	return nil
}

func testRequestCollapse(ctx context.Context) error {
	r := getTestReq("", &query{wait: 1 * time.Second}, nil)

	var beforeSendCount int
	var mu sync.Mutex

	r.CacheOptions.BeforeSend = func(r *fsthttp.Request) error {
		mu.Lock()
		beforeSendCount++
		mu.Unlock()
		r.Header.Set("x-count", "COUNT "+strconv.Itoa(beforeSendCount))
		return nil
	}

	r2 := r.Clone()

	ch := make(chan struct{})
	errch := make(chan error, 2)

	var resp *fsthttp.Response
	var resp2 *fsthttp.Response

	go func() {
		<-ch
		var err error
		resp, err = r.Send(ctx, backend)
		time.Sleep(10 * time.Millisecond)
		errch <- err
	}()

	go func() {
		<-ch
		var err error
		resp2, err = r2.Send(ctx, backend)
		time.Sleep(20 * time.Millisecond)
		errch <- err
	}()

	// start the goroutines
	close(ch)

	for i := 0; i < 2; i++ {
		err := <-errch
		if err != nil {
			return fmt.Errorf("error during send: %v", err)
		}
	}

	// check beforeSend only executed once
	mu.Lock()
	if got, want := beforeSendCount, 1; got != want {
		mu.Unlock()
		return fmt.Errorf("beforeSendCount=%v, want %v", got, want)
	}
	mu.Unlock()

	// check the requests were collapsed by each request having the same x-count header value
	var responses = []*fsthttp.Response{resp, resp2}
	for i, rs := range responses {
		j := decodeBody(rs.Body)
		if got, want := j.Headers["x-count"], "COUNT 1"; got != want {
			return fmt.Errorf("resp %v headers[x-count]=%v, want %v", i, got, want)
		}
	}

	if !resp.FromCache() && !resp2.FromCache() {
		return fmt.Errorf("neither request was cached: %v/%v", resp.FromCache(), resp2.FromCache())
	}

	if resp.FromCache() && resp2.FromCache() {
		return fmt.Errorf("both requests were cached: %v/%v", resp.FromCache(), resp2.FromCache())
	}

	return nil
}

func testRequestCollapseThree(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var beforeSendCount int

	r.CacheOptions.BeforeSend = func(r *fsthttp.Request) error {
		beforeSendCount++
		r.Header.Set("x-count", "COUNT "+strconv.Itoa(beforeSendCount))
		return nil
	}

	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		// slow backend
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	r2 := r.Clone()
	r3 := r.Clone()

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send: %v", err)
	}

	resp2, err := r2.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send 2: %v", err)
	}

	resp3, err := r3.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send 3: %v", err)
	}

	// check beforeSend only executed once
	if got, want := beforeSendCount, 1; got != want {
		return fmt.Errorf("beforeSendCount=%v, want %v", got, want)
	}

	// check the requests were collapsed by each request having the same x-count header value
	var responses = []*fsthttp.Response{resp, resp2, resp3}
	for i, r := range responses {
		j := decodeBody(r.Body)
		if got, want := j.Headers["x-count"], "COUNT 1"; got != want {
			return fmt.Errorf("resp %v headers[x-count]=%v, want %v", i, got, want)
		}
	}

	if c1, c2, c3 := resp.FromCache(), resp2.FromCache(), resp3.FromCache(); c1 || !c2 || !c3 {
		return fmt.Errorf("wrong request was cached: c1=%v, c2=%v c3=%v", c1, c3, c3)
	}

	return nil
}

func testRequestCollapseUncacheable(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var beforeSendCount int

	r.CacheOptions.BeforeSend = func(r *fsthttp.Request) error {
		beforeSendCount++
		r.Header.Set("x-count", "COUNT "+strconv.Itoa(beforeSendCount))
		return nil
	}

	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		r.SetUncacheableDisableCollapsing()
		return nil
	}

	r2 := r.Clone()
	r3 := r.Clone()
	r4 := r.Clone()

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send: %v", err)
	}
	resp2, err := r2.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send 2: %v", err)
	}

	resp3, err := r3.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send 3: %v", err)
	}

	if got, want := beforeSendCount, 3; got != want {
		return fmt.Errorf("beforeSendCount=%v, want %v", got, want)
	}

	// check that none of the responses were from the cache
	var responses = []*fsthttp.Response{resp, resp2, resp3}
	for i, r := range responses {
		if r.FromCache() {
			return fmt.Errorf("cached uncacheable: i=%v", i)

		}
	}

	// next request after a small delay
	time.Sleep(50 * time.Millisecond)

	resp4, err := r4.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send 4: %v", err)
	}

	if resp4.FromCache() {
		return fmt.Errorf("cached uncacheable 4")

	}

	return nil
}

func testRequestCollapseVary(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var beforeSendCount int

	r.CacheOptions.BeforeSend = func(r *fsthttp.Request) error {
		beforeSendCount++
		return nil
	}

	const headerUA = "user-agent"

	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		r.SetVary(headerUA)
		r.SetCacheable()
		return nil
	}

	r2 := r.Clone()
	r3 := r.Clone()
	r4 := r.Clone()

	r.Header.Set(headerUA, "bot1")
	r2.Header.Set(headerUA, "bot1")

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send: %v", err)
	}
	resp2, err := r2.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send 2: %v", err)
	}

	if got, want := beforeSendCount, 1; got != want {
		return fmt.Errorf("beforeSendCount=%v, want %v", got, want)
	}

	if c1, c2 := resp.FromCache(), resp2.FromCache(); c1 || !c2 {
		return fmt.Errorf("bot1 collapse cached: got c1=%v c2=%v", c1, c2)
	}

	r3.Header.Set(headerUA, "bot2")
	r4.Header.Set(headerUA, "bot2")

	resp3, err := r3.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send 3: %v", err)
	}

	resp4, err := r4.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("error during send 4: %v", err)
	}

	if got, want := beforeSendCount, 2; got != want {
		return fmt.Errorf("beforeSendCount=%v, want %v", got, want)
	}

	if c3, c4 := resp3.FromCache(), resp4.FromCache(); c3 || !c4 {
		return fmt.Errorf("bot2 collapse cached: got c3=%v c4=%v", c3, c4)
	}

	return nil
}

func testAfterSendError(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var errAfterSend = errors.New("after send error")

	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		return errAfterSend
	}

	_, err := r.Send(ctx, backend)
	if !errors.Is(err, errAfterSend) {
		return fmt.Errorf("unexpected error: got %v, want %v", err, errAfterSend)
	}

	return nil
}

func testAfterSendCandidateResponsePropertiesUncached(ctx context.Context) error {

	r := getTestReq("", nil, nil)

	var calledAfterSend bool

	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true

		//	   strictEqual(candidateRes.cached, false);

		if b, err := r.IsStale(); b || err != nil {
			return fmt.Errorf("candidate is stale=%v or err=%v, want false, nil", b, err)
		}

		if age, err := r.Age(); age != 0 || err != nil {
			return fmt.Errorf("candidate has age=%v or err=%v, want 0, nil", age, err)
		}

		if ttl, err := r.TTL(); ttl != 3600 || err != nil {
			return fmt.Errorf("candidate has ttl=%v or err=%v, want 3600, nil", ttl, err)
		}

		if got, err := r.Vary(); err != nil {
			return fmt.Errorf("candidate.vary has err=%v, want nil", err)
		} else if want := "accept-encoding"; got != want {
			return fmt.Errorf("candidate.vary=%v, want %v", got, want)
		}

		if surrogate, err := r.SurrogateKeys(); err != nil {
			return fmt.Errorf("candidate has err=%v, want nil", err)
		} else if got, want := len(surrogate), 64; got != want {
			return fmt.Errorf("candidate has len(surrogate)=%v, want %v", got, want)
		}

		r.SetUncacheable()

		return nil
	}

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	//   strictEqual(res.cached, false);

	if got, ok := resp.StaleWhileRevalidate(); !ok || got != 0 {
		return fmt.Errorf("response.stale()=%v,%v want %v, %v", got, ok, 0, true)
	}

	if got, ok := resp.Age(); !ok || got != 0 {
		return fmt.Errorf("response.age()=%v,%v want %v, %v", got, ok, 0, true)
	}

	if got, ok := resp.TTL(); !ok || got != 3600 {
		return fmt.Errorf("response.ttl()=%v, %v, want %v, %v", got, ok, 3600, true)
	}

	if got, want := resp.Vary(), "accept-encoding"; got != want {
		return fmt.Errorf("response.Vary()=%v, want %v", got, want)
	}

	if got, wantLen := resp.SurrogateKeys(), 10; len(got) < wantLen {
		return fmt.Errorf("response.SurrogateKeys()=%v, want length %v", got, wantLen)
	}

	return nil
}

func testAfterSendCandidateResponsePropertiesCached(ctx context.Context) error {

	r := getTestReq("", nil, nil)

	var calledAfterSend bool

	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true

		//	   strictEqual(candidateRes.cached, false);
		/*

			if b, err := r.IsStale(); b || err != nil {
				return fmt.Errorf("candidate is stale=%v or err=%v, want false, nil", b, err)
			}

			if age, err := r.GetAge(); age != 0 || err != nil {
				return fmt.Errorf("candidate has age=%v or err=%v, want 0, nil", age, err)
			}

			if ttl, err := r.GetTTL(); ttl != time.Duration(0) || err != nil {
				return fmt.Errorf("candidate has ttl=%v or err=%v, want 0, nil", ttl, err)
			}

			// TODO(dgryski): This doesn't match javascript test but I don't know expected behaviour; fixed for now
			if got, err := r.GetVary(); err != nil {
				return fmt.Errorf("candidate.vary has err=%v, want nil", err)
			} else if want := "accept-encoding"; !reflect.DeepEqual(got, want) {
				return fmt.Errorf("candidate.vary=%v, want %v", got, want)
			}

			if surrogate, err := r.GetSurrogateKeys(); err != nil {
				return fmt.Errorf("candidate has err=%v, want nil", err)
			} else if got, want := len(surrogate), 64; got != want {
				return fmt.Errorf("candidate has len(surrogate)=%v, want %v", got, want)
			}

		*/

		r.SetCacheable()

		return nil
	}

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	// TODO(dgryski): figure out some of these fields

	/*
	   strictEqual(res.cached, false);
	   strictEqual(res.stale, false);
	   strictEqual(res.ttl, 3600);
	   strictEqual(res.age, 0);
	   deepStrictEqual(res.vary, []);
	   strictEqual(res.surrogateKeys.length, 1);
	   strictEqual(typeof res.surrogateKeys[0], 'string');
	   strictEqual(res.surrogateKeys[0].length > 10, true);
	*/

	if got, ok := resp.Age(); ok || got != 0 {
		return fmt.Errorf("response.age()=%v,%v want %v, %v", got, ok, got, false)
	}

	// TODO(dgryski): This is also still broken -- doesn't match JS code
	if got, ok := resp.TTL(); ok || got != 0 {
		return fmt.Errorf("response.ttl()=%v, %v, want %v, %v", got, ok, 0, true)
	}

	if got, want := resp.Vary(), "accept-encoding"; got != want {
		return fmt.Errorf("response.Vary()=%v, want %v", got, want)
	}

	if got, wantLen := resp.SurrogateKeys(), 10; len(got) < wantLen {
		return fmt.Errorf("response.SurrogateKeys()=%v, want length > %v", got, wantLen)
	}

	return nil
}

func testAfterSendNoCache(ctx context.Context) error {

	r := getTestReq("", nil, nil)

	var calledAfterSend bool

	const headerKey = "cache-control"
	const headerVal = "private, no-store"

	afterSend := func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true

		r.SetHeader(headerKey, headerVal)
		age, _ := r.Age()
		r.SetTTL(3600 - age)

		r.SetUncacheable()

		return nil

	}

	r.CacheOptions.AfterSend = afterSend

	r2 := r.Clone()

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	if got, want := resp.Header.Get(headerKey), headerVal; got != want {
		return fmt.Errorf("resp.Header.Get(%v)=%v, want %v", headerKey, got, want)
	}

	calledAfterSend = false

	resp2, err := r2.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend2=false")
	}

	if got, want := resp2.Header.Get(headerKey), headerVal; got != want {
		return fmt.Errorf("resp2.Header.Get(%v)=%v, want %v", headerKey, got, want)
	}

	if got, want := resp2.Header.Get("x-cache"), "MISS"; got != want {
		return fmt.Errorf("resp2.Header.Get(%v)=%v, want %v", "x-cache", got, want)
	}

	return nil
}

func testAfterSendCached(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var calledAfterSend bool

	const headerKey = "cache-control"
	const headerVal = "private, no-store"

	afterSend := func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true

		r.SetHeader(headerKey, headerVal)
		age, _ := r.Age()
		r.SetTTL(3600 - age)

		r.SetCacheable()

		return nil
	}

	r.CacheOptions.AfterSend = afterSend

	r2 := r.Clone()

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	if got, want := resp.Header.Get(headerKey), headerVal; got != want {
		return fmt.Errorf("resp.Header.Get(%v)=%v, want %v", headerKey, got, want)
	}

	calledAfterSend = false

	r2.CacheOptions.AfterSend = afterSend

	resp2, err := r2.Send(ctx, backend)
	if err != nil {
		return err
	}

	if calledAfterSend {
		return errors.New("calledAfterSend2=true: should be false for cache hit")
	}

	if got, want := resp2.Header.Get(headerKey), headerVal; got != want {
		return fmt.Errorf("resp2.Header.Get(%v)=%v, want %v", headerKey, got, want)
	}

	if got, want := resp2.Header.Get("x-cache"), "HIT"; got != want {
		return fmt.Errorf("resp2.Header.Get(%v)=%v, want %v", "x-cache", got, want)
	}

	return nil
}

func testAfterSendExpire(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var calledAfterSend bool

	const headerKey = "cache-control"
	const headerVal = "max-age=2"

	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true
		r.SetHeader(headerKey, headerVal)
		r.SetCacheable()
		return nil
	}

	r2 := r.Clone()
	r3 := r.Clone()

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	if got, want := resp.Header.Get("x-cache"), "MISS"; got != want {
		return fmt.Errorf("resp.Header.Get(%v)=%v, want %v", "x-cache", got, want)
	}

	if got, want := resp.Header.Get("cache-control"), "max-age=2"; got != want {
		return fmt.Errorf("resp.Header.Get(%v)=%v, want %v", "cache-control", got, want)
	}

	calledAfterSend = false

	r2.CacheOptions.AfterSend = func(CandidateResponse *fsthttp.CandidateResponse) error {
		calledAfterSend = true
		return nil
	}

	// should still be cached
	time.Sleep(500 * time.Millisecond)

	resp2, err := r2.Send(ctx, backend)
	if err != nil {
		return err
	}

	if calledAfterSend {
		return errors.New("calledAfterSend2=true: should be false for cache hit")
	}

	if age, ok := resp2.Age(); !ok || age > 2 {
		return fmt.Errorf("resp2.age = %v, want < 2", age)
	}

	if got, want := resp2.Header.Get("x-cache"), "HIT"; got != want {
		return fmt.Errorf("resp2.Header.Get(%v)=%v, want %v", "x-cache", got, want)
	}

	if got, want := resp2.Header.Get("cache-control"), "max-age=2"; got != want {
		return fmt.Errorf("resp2.Header.Get(%v)=%v, want %v", "cache-control", got, want)
	}

	// expire the cache
	time.Sleep(2 * time.Second)

	calledAfterSend = false
	r3.CacheOptions.AfterSend = func(CandidateResponse *fsthttp.CandidateResponse) error {
		calledAfterSend = true
		return nil
	}

	resp3, err := r3.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend3=false: should be true for cache miss")
	}

	if got, ok := resp3.Age(); ok || got != 0 {
		return fmt.Errorf("resp3.age = %v,%v want %v", got, ok, 0)
	}

	if got, want := resp3.Header.Get("x-cache"), "MISS"; got != want {
		return fmt.Errorf("resp3.Header.Get(%v)=%v, want %v", "x-cache", got, want)
	}

	return nil
}

type toUpper struct {
	r io.ReadCloser
}

func (t toUpper) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	copy(p, bytes.ToUpper(p))
	return n, err
}

func (t toUpper) Close() error {
	return t.r.Close()
}

func testAfterSendBodyTransform(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var calledAfterSend bool
	var calledBodyTransform bool

	afterSend := func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true

		r.SetBodyTransform(func(r io.ReadCloser) io.ReadCloser {
			calledBodyTransform = true
			return toUpper{r}
		})

		r.SetUncacheable()

		return nil
	}

	r.CacheOptions.AfterSend = afterSend

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	if !calledBodyTransform {
		return errors.New("calledBodyTransform=false")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("readall(body)=%v, want %v", err, nil)
	}

	if !bytes.Contains(body, []byte(`"URL": "/ANYTHING/`)) {
		return fmt.Errorf("body not uppercase: got %v", string(body))
	}

	return nil
}

var errReaderErr = errors.New("error reader: read")

type errReader struct{}

func (e errReader) Read(p []byte) (int, error) {
	return 0, errReaderErr
}

func (e errReader) Close() error {
	return nil
}

func testAfterSendBodyTransformFailedCacheable(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var calledAfterSend bool

	afterSend := func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true

		r.SetBodyTransform(func(r io.ReadCloser) io.ReadCloser {
			return errReader{}
		})

		r.SetCacheable()

		return nil
	}

	r.CacheOptions.AfterSend = afterSend

	_, err := r.Send(ctx, backend)
	if !errors.Is(err, errReaderErr) {
		return fmt.Errorf("r.Send() err=%v, want %v", err, errReaderErr)
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	return nil
}

func testAfterSendBodyTransformFailedUncacheable(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var calledAfterSend bool

	afterSend := func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true

		r.SetBodyTransform(func(r io.ReadCloser) io.ReadCloser {
			return errReader{}
		})

		r.SetUncacheableDisableCollapsing()
		return nil
	}

	r.CacheOptions.AfterSend = afterSend

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("r.Send() err=%v, want %v", err, nil)
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	if _, err := io.ReadAll(resp.Body); !errors.Is(err, errReaderErr) {
		return fmt.Errorf("resp.Body.Read() err=%v, want %v", err, errReaderErr)
	}

	return nil
}

func testAfterSendMutations(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	r.Header.Set("fastly-debug", "1")

	var calledAfterSend bool
	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true
		r.SetStatus(201)
		r.SetHeader("x-custom", "test")
		r.SetTTL(4000)
		r.SetStaleWhileRevalidate(400)
		r.SetSensitive(true)
		r.SetSurrogateKeys("key1,key2")
		r.SetVary("accept,user-agent")
		r.SetCacheable()
		return nil
	}

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	if got, want := resp.StatusCode, 201; got != want {
		return fmt.Errorf("response.StatusCode=%v want %v", got, want)
	}

	if got, want := resp.Header.Get("x-custom"), "test"; got != want {
		return fmt.Errorf("r.Header.Get('x-custom'): got %q, want %q", got, want)
	}

	// TODO(dgryski): javascript allow this, but rust doesn't
	/*
		if got, ok := resp.StaleWhileRevalidate(); !ok || got != 400 {
			return fmt.Errorf("response.stale()=%v,%v want %v, %v", got, ok, 400, true)
		}
		if got, ok := resp.TTL(); !ok || got != 4000 {
			return fmt.Errorf("response.ttl()=%v, %v, want %v, %v", got, ok, 4000, true)
		}
	*/

	if got, want := resp.Vary(), "accept,user-agent"; got != want {
		return fmt.Errorf("response.Vary()=%q, want %q", got, want)
	}

	if got, want := resp.SurrogateKeys(), "key1,key2"; got != want {
		return fmt.Errorf("response.SurrogateKeys()=%q, want %q", got, want)
	}

	return nil
}

func testAfterSendHeaderRemove(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var calledAfterSend bool
	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true
		r.SetHeader("custom", "custom-header")
		r.DelHeader("access-control-allow-origin")
		r.DelHeader("access-control-allow-credentials")
		r.SetUncacheableDisableCollapsing()
		return nil
	}

	resp, err := r.Send(ctx, backend)
	if err != nil {
		return err
	}

	if !calledAfterSend {
		return errors.New("calledAfterSend=false")
	}

	if got, want := resp.Header.Get("custom"), "custom-header"; got != want {
		return fmt.Errorf("r.Header.Get('custom'): got %v, want %v", got, want)
	}

	if got, want := resp.Header.Get("access-control-allow-origin"), ""; got != want {
		return fmt.Errorf("r.Header.Get('access-control-allow-origin'): got %v, want %v", got, want)
	}

	return nil
}

func testAfterSendPass(ctx context.Context) error {
	r := getTestReq("", nil, nil)

	var calledAfterSend bool

	r.CacheOptions.Pass = true
	r.CacheOptions.AfterSend = func(r *fsthttp.CandidateResponse) error {
		calledAfterSend = true
		return nil
	}

	_, err := r.Send(ctx, backend)
	if err != nil {
		return fmt.Errorf("unexpected Send error: %v", err)
	}
	if calledAfterSend {
		return fmt.Errorf("unexpected after send called, want false")
	}

	return nil
}
