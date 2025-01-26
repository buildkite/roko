package roko_test

import (
	"context"
	"fmt"
	"time"

	"github.com/buildkite/roko/v2"
)

func ExampleBackoff() {
	ctx := context.Background()
	// With jitter (makes output random):
	// seq := roko.Limit(5, roko.Jitter(roko.Exp(1*time.Millisecond, 1.1)))
	// Without jitter:
	seq := roko.Limit(5, roko.Exp(1*time.Millisecond, 1.1))
	for i, nw := range roko.Backoff(ctx, seq) {
		fmt.Printf("Iteration %d: ", i)
		if nextWait := *nw; nextWait < 0 {
			fmt.Println("Last try!")
		} else {
			fmt.Println("Next wait =", nextWait)
		}
	}
	// Output:
	// Iteration 0: Next wait = 1ms
	// Iteration 1: Next wait = 1.1ms
	// Iteration 2: Next wait = 1.21ms
	// Iteration 3: Next wait = 1.331ms
	// Iteration 4: Next wait = 1.4641ms
	// Iteration 5: Last try!
}

func ExampleBackoff_override() {
	ctx := context.Background()

	seq := roko.Limit(5, roko.Exp(1*time.Millisecond, 1.1))
	for i, nw := range roko.Backoff(ctx, seq) {
		fmt.Printf("Iteration %d, next wait duration %v\n", i, *nw)
		if i == 3 {
			*nw = 100 * time.Millisecond
			fmt.Printf("Overriding the wait time to be %v\n", *nw)
		}
	}
	// Output:
	// Iteration 0, next wait duration 1ms
	// Iteration 1, next wait duration 1.1ms
	// Iteration 2, next wait duration 1.21ms
	// Iteration 3, next wait duration 1.331ms
	// Overriding the wait time to be 100ms
	// Iteration 4, next wait duration 1.4641ms
	// Iteration 5, next wait duration -1ns
}
