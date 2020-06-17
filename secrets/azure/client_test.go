package azuresecrets

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	t.Parallel()
	t.Run("First try success", func(t *testing.T) {
		_, err := retry(context.Background(), func() (interface{}, bool, error) {
			return nil, true, nil
		})
		assertErrorIsNil(t, err)
	})

	t.Run("Three retries", func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		count := 0

		_, err := retry(context.Background(), func() (interface{}, bool, error) {
			count++
			if count >= 3 {
				return nil, true, nil
			}
			return nil, false, nil
		})
		equal(t, count, 3)

		// each sleep can last from 2 to 8 seconds
		elapsed := time.Now().Sub(start).Seconds()
		if elapsed < 4 || elapsed > 16 {
			t.Fatalf("expected time of 4-16 seconds, got: %f", elapsed)
		}
		assertErrorIsNil(t, err)
	})

	t.Run("Error on attempt", func(t *testing.T) {
		t.Parallel()
		_, err := retry(context.Background(), func() (interface{}, bool, error) {
			return nil, true, errors.New("Fail")
		})
		if err == nil || !strings.Contains(err.Error(), "Fail") {
			t.Fatalf("expected failure error, got: %v", err)
		}
	})

	// timeout test
	t.Run("Timeout", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping test in short mode.")
		}
		t.Parallel()
		start := time.Now()

		timeout := 10 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		called := 0
		_, err := retry(ctx, func() (interface{}, bool, error) {
			called++
			return nil, false, nil
		})
		elapsed := time.Now().Sub(start)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if called == 0 {
			t.Fatalf("retryable function was never called")
		}
		assertDuration(t, elapsed, timeout, 100*time.Millisecond)
	})

	t.Run("Cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(7 * time.Second)
			cancel()
		}()

		start := time.Now()
		_, err := retry(ctx, func() (interface{}, bool, error) {
			return nil, false, nil
		})
		elapsed := time.Now().Sub(start).Seconds()
		if elapsed < 6 || elapsed > 8 {
			t.Fatalf("expected time of ~7 seconds, got: %f", elapsed)
		}

		if err == nil {
			t.Fatalf("expected err: got nil")
		}
		underlyingErr := errors.Unwrap(err)
		if underlyingErr != context.Canceled {
			t.Fatalf("expected %s, got: %v", context.Canceled, err)
		}
	})
}

// assertDuration with a certain amount of flex in the exact value
func assertDuration(t *testing.T, actual, expected, delta time.Duration) {
	t.Helper()

	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}

	if diff > delta {
		t.Fatalf("Actual duration %s does not equal expected %s with delta %s", actual, expected, delta)
	}
}
