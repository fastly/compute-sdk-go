// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	begin := time.Now()
	rand.Seed(begin.UnixNano())

	c := make(chan string, 5)
	for i := 0; i < cap(c); i++ {
		go func(i int) {
			r := 1 + rand.Intn(99)
			d := time.Duration(r) * time.Millisecond
			time.Sleep(d)
			c <- fmt.Sprintf("goroutine %d took %s", i, d)
		}(i)
	}

	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		for i := 0; i < cap(c); i++ {
			fmt.Fprintln(w, <-c)
		}
		fmt.Fprintln(w, "overall", time.Since(begin))
	})
}
