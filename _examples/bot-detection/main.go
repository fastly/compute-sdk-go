// Copyright 2022 Fastly, Inc.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		bot, _ := r.BotDetection()

		if bot.Analyzed {
			if bot.Detected {
				log.Println(w, "request from bot:", bot.Category, bot.Name)
			}
		}

		fmt.Fprintf(w, "Hello, %s!\n", r.RemoteAddr)
	})
}
