package main

import "time"

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		return 0
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<uint(attempt-1)) * time.Second
}

func shouldRetry(status int, attempt int) bool { return status >= 500 && attempt < 5 }
