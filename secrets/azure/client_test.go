package azuresecrets

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	t.Parallel()
	t.Run("First try success", func(t *testing.T) {
		_, err := retry(context.Background(), func() (interface{}, bool, error) {
			return nil, true, nil
		})
		nilErr(t, err)
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
		nilErr(t, err)
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
		_, err := retry(context.Background(), func() (interface{}, bool, error) {
			return nil, false, nil
		})
		elapsed := time.Now().Sub(start).Seconds()
		if elapsed < 178 || elapsed > 182 {
			t.Fatalf("expected time of ~180 seconds, got: %f", elapsed)
		}
		if err == nil || !strings.Contains(err.Error(), "timeout") {
			t.Fatalf("expected timeout error, got: %v", err)
		}
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

		if err == nil || !strings.Contains(err.Error(), "cancelled") {
			t.Fatalf("expected cancelled error, got: %v", err)
		}
	})
}
