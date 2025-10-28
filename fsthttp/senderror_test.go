// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

func TestSendErrorWrapped(t *testing.T) {
	detail := SendError{
		Tag: SendErrorConnectionTimeout,
	}

	wrappedErr := fastly.FastlyError{
		Status: fastly.FastlyStatusError,
		Detail: detail,
	}

	wrappedAgainErr := fmt.Errorf("request failed: %w", wrappedErr)

	var serr SendError
	if !errors.As(wrappedAgainErr, &serr) {
		t.Fatal("errors.As() failed to extract SendError")
	}

	if serr.Cause() != SendErrorConnectionTimeout {
		t.Errorf("Cause() = %v, want %v", serr.Cause(), SendErrorConnectionTimeout)
	}
}
