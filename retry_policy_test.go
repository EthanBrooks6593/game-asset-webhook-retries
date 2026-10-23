package main

import (
	"testing"
	"time"
)

func TestRetryDecisionForModerationDelivery(t *testing.T) {
	tests := []struct {
		name            string
		status, attempt int
		retry           bool
		delay           time.Duration
	}{
		{"first server failure", 503, 1, true, time.Second},
		{"last allowed attempt", 503, 5, false, 16 * time.Second},
		{"client rejection", 400, 1, false, time.Second},
		{"success", 200, 1, false, time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetry(tt.status, tt.attempt); got != tt.retry {
				t.Fatalf("retry=%v, want %v", got, tt.retry)
			}
			if got := retryDelay(tt.attempt); got != tt.delay {
				t.Fatalf("delay=%v, want %v", got, tt.delay)
			}
		})
	}
}
