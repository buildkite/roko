package roko

import (
	"iter"
	"math/rand/v2"
	"time"
)

// Exp returns an iterator over a geometric sequence of pause
// durations (i.e. an exponential backoff):
//
//	{initial, initial*factor, initial*factor*factor, initial*factor^3, ...}
func Exp(initial time.Duration, factor float64) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		cur := initial
		for yield(cur) {
			cur = time.Duration(float64(cur) * factor)
		}
	}
}

// Const returns an iterator over a constant sequence.
func Const(dur time.Duration) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for yield(dur) {
		}
	}
}

// Jitter multiplies each duration in the input sequence by a random
// variable X ~ U[0,1] (i.e. for each input duration d, the corresponding output
// duration will be a random value in the range [0, d]).
func Jitter(seq iter.Seq[time.Duration]) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for nw := range seq {
			if !yield(rand.N(nw)) {
				return
			}
		}
	}
}

// IntervalJitter adds to each duration in the input sequence a random variable
// X ~ U[lo,hi] (i.e. for each input duration d, the corresponding output
// duration will be a random value in the range [d+lo, d+hi]). However, negative
// durations will be clamped up to 0.
func IntervalJitter(lo, hi time.Duration, seq iter.Seq[time.Duration]) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for nw := range seq {
			nw += lo + rand.N(hi-lo)
			if !yield(max(nw, 0)) {
				return
			}
		}
	}
}

// FactorJitter multiplies each duration in the input sequence by a random
// variable X ~ U[lo,hi] (i.e. for each input duration d, the corresponding
// output duration will be a random value in the range [d*lo, d*hi]).
// Negative durations will be clamped up to 0.
func FactorJitter(lo, hi float64, seq iter.Seq[time.Duration]) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for nw := range seq {
			f := lo + rand.Float64()*(hi-lo)
			nw := time.Duration(float64(nw) * f)
			if !yield(max(nw, 0)) {
				return
			}
		}
	}
}

// Limit returns an iterator that yields the first n items from the input
// sequence.
func Limit(n int, seq iter.Seq[time.Duration]) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for nw := range seq {
			if n <= 0 {
				return
			}
			n--
			if !yield(nw) {
				return
			}
		}
	}
}

// Concat returns an iterator that yields values from each sequence until it
// is exhausted.
func Concat(seqs ...iter.Seq[time.Duration]) iter.Seq[time.Duration] {
	return func(yield func(time.Duration) bool) {
		for _, s := range seqs {
			for nw := range s {
				if !yield(nw) {
					return
				}
			}
		}
	}
}
