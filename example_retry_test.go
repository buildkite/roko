package roko_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/buildkite/roko/v2"
)

func ExampleRetry0() {
	ctx := context.Background()
	seq := roko.Limit(5, roko.Exp(1*time.Millisecond, 1.1))
	err := roko.Retry0(ctx, seq, func(i int, nw *time.Duration) error {
		// Pretend to be some function that errors sometimes
		if i < 3 {
			return errors.New("spurious error")
		}
		return nil
	})

	fmt.Println(err)
	// Output: <nil>
}

func ExampleRetry1() {
	ctx := context.Background()
	seq := roko.Limit(5, roko.Exp(1*time.Millisecond, 1.1))
	answer, err := roko.Retry1(ctx, seq, func(i int, nw *time.Duration) (int, error) {
		// Pretend to be some function that errors sometimes
		if i < 3 {
			return -1, errors.New("spurious error")
		}
		return 42, nil
	})

	fmt.Println(answer, err)
	// Output: 42 <nil>
}

func ExampleRetry1_unrecoverable() {
	ctx := context.Background()
	seq := roko.Limit(5, roko.Exp(1*time.Millisecond, 1.1))
	answer, err := roko.Retry1(ctx, seq, func(i int, nw *time.Duration) (int, error) {
		// Pretend to be some function that errors sometimes
		if i < 3 {
			return -1, errors.New("spurious error")
		}
		if i == 3 {
			return -2, fmt.Errorf("%w catastrophic error", roko.ErrUnrecoverable)
		}
		return 42, nil
	})

	fmt.Println(answer, err)
	// Output: -2 unrecoverable catastrophic error
}

func ExampleRetry0_sentinel_next_wait() {
	ctx := context.Background()
	seq := roko.Limit(5, roko.Exp(1*time.Millisecond, 1.1))

	err := roko.Retry0(ctx, seq, func(i int, nw *time.Duration) error {
		// Pretend to be some function that errors sometimes.
		fmt.Printf("Iteration %d\n", i)
		if i == 3 {
			*nw = roko.SentinelDuration
			return errors.New("some other error")
		}
		return errors.New("spurious error")
	})

	fmt.Println(err)
	// Output:
	// Iteration 0
	// Iteration 1
	// Iteration 2
	// Iteration 3
	// some other error
}
