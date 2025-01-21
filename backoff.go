package roko

import (
	"context"
	"iter"
	"time"
)

// SentinelDuration is yielded for the last iteration of a backoff. Note that
// *any* negative duration yielded from a sequence (or used as an override) will
// terminate a backoff loop, not only this particular value.
const SentinelDuration = time.Duration(-1)

// Backoff returns an iterator that pauses according to the pause durations
// from pauseSeq. It always yields immediately, then (if the input sequence
// hasn't ended), waits for that duration before yielding again. A negative
// pause duration is yielded for the final iteration, with no pause afterwards.
// Backoff yields a pointer to the next wait duration, allowing the caller
// to override the wait time following the current iteration.
func Backoff(ctx context.Context, pauseSeq iter.Seq[time.Duration]) iter.Seq2[int, *time.Duration] {
	return func(yield func(int, *time.Duration) bool) {
		i := 0
		for nw := range appendSentiel(pauseSeq) {
			if !yield(i, &nw) {
				return
			}
			if nw < 0 {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(nw):
				// Done waiting! Onto the next iteration.
			}
			i++
		}
	}
}

// appendSentiel is an internal utility that, after the input sequence is
// exhausted, yields SentinelDuration once to signal the final iteration of
// the backoff loop.
func appendSentiel(seq iter.Seq[time.Duration]) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for nw := range seq {
			if !yield(nw) {
				return
			}
		}
		yield(SentinelDuration)
	}
}
