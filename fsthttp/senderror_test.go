// Copyright 2022 Fastly, Inc.

package fsthttp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fastly/compute-sdk-go/internal/abi/fastly"
)

func TestSendErrorIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "connection timeout matches",
			err:      ErrConnectionTimeout,
			target:   ErrConnectionTimeout,
			expected: true,
		},
		{
			name:     "different errors don't match",
			err:      ErrConnectionTimeout,
			target:   ErrDNSTimeout,
			expected: false,
		},
		{
			name:     "wrapped error matches",
			err:      fmt.Errorf("send failed: %w", ErrConnectionTimeout),
			target:   ErrConnectionTimeout,
			expected: true,
		},
		{
			name:     "FastlyError with SendErrorDetail matches",
			err:      fastly.FastlyError{Status: fastly.FastlyStatusError, Detail: fastly.SendErrorConnectionTimeout},
			target:   ErrConnectionTimeout,
			expected: true,
		},
		{
			name:     "doubly wrapped error matches",
			err:      fmt.Errorf("request failed: %w", fmt.Errorf("backend error: %w", ErrDNSError)),
			target:   ErrDNSError,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err, tt.target)
			if result != tt.expected {
				t.Errorf("errors.Is() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSendErrorAs(t *testing.T) {
	dnsErr := fastly.SendErrorDNSError

	wrappedErr := fastly.FastlyError{
		Status: fastly.FastlyStatusError,
		Detail: dnsErr,
	}

	wrappedAgainErr := fmt.Errorf("request failed: %w", wrappedErr)

	// Test that we can extract it with errors.As()
	var se SendError
	if !errors.As(wrappedAgainErr, &se) {
		t.Fatal("errors.As() failed to extract SendError")
	}

	// Verify it matches the expected error
	if !errors.Is(se, ErrDNSError) {
		t.Error("extracted SendError does not match ErrDNSError")
	}
}
