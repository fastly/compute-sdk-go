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
		bot, err := r.BotDetection()
		if err != nil {
			// Log the failure but don't return; gracefully handle the lack of bot detection result instead.
			// bot.Analyzed will be false.
			log.Println("Bot detection not available:", err)
		}

		if bot.Analyzed {
			if bot.Detected {
				log.Println(w, "request from bot:", bot.Category, bot.Name)
			}
		}

		fmt.Fprintf(w, "Hello, %s!\n", r.RemoteAddr)
	})
}
