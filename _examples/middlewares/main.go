// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	var h fsthttp.Handler
	h = newHandler(fmt.Sprintf("Hello from %s!", os.Getenv("FASTLY_POP")))
	h = newLoggingMiddleware(h)
	fsthttp.Serve(h)
}

//
//
//

type handler struct {
	greeting string
}

func newHandler(greeting string) *handler {
	return &handler{
		greeting: greeting,
	}
}

func (h *handler) ServeHTTP(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
	fmt.Fprintln(w, h.greeting)
}

//
//
//

type loggingMiddleware struct {
	next fsthttp.Handler
}

func newLoggingMiddleware(next fsthttp.Handler) *loggingMiddleware {
	return &loggingMiddleware{
		next: next,
	}
}

func (mw *loggingMiddleware) ServeHTTP(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
	irw := &interceptingResponseWriter{ResponseWriter: w, code: fsthttp.StatusOK}

	defer func(begin time.Time) {
		log.Printf("%s: %s %s %s: %d (%s)", r.RemoteAddr, r.Proto, r.Method, r.URL, irw.code, time.Since(begin))
	}(time.Now())

	mw.next.ServeHTTP(ctx, irw, r)
}

//
//
//

type interceptingResponseWriter struct {
	fsthttp.ResponseWriter
	code int
}

func (irw *interceptingResponseWriter) WriteHeader(code int) {
	irw.code = code
	irw.ResponseWriter.WriteHeader(code)
}
