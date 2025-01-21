package roko

import (
	"context"
	"errors"
	"iter"
	"time"
)

// ErrUnrecoverable is a sentinel error value that can be returned (wrapped) from a
// function to signal that any retry loop should be aborted immediately.
const ErrUnrecoverable = unrecoverableErr("unrecoverable")

type unrecoverableErr string

func (e unrecoverableErr) Error() string { return string(e) }

// Retry0 retries a function if it returns a non-nil error, with pauses between
// retries taken from pauseSeq. The function is passed the iteration index
// and a pointer to the next wait duration. The function can end the retry loop
// early in two ways:
// - return ErrUnrecoverable (or an error wrapping ErrUnrecoverable), or
// - override the wait duration to SentinelDuration (with any return value).
func Retry0(ctx context.Context, pauseSeq iter.Seq[time.Duration], f func(int, *time.Duration) error) error {
	var err error
	for i, nw := range Backoff(ctx, pauseSeq) {
		err = f(i, nw)
		if errors.Is(err, ErrUnrecoverable) {
			return err
		}
		if err != nil {
			continue
		}
		return nil
	}
	return err
}

// Retry1 retries a function if it returns a non-nil error, with pauses between
// retries taken from pauseSeq. It returns the values from the last call to f.
// The function is passed the iteration index and a pointer to the next wait
// duration. The function can end the retry loop early in two ways:
// - return ErrUnrecoverable (or an error wrapping ErrUnrecoverable), or
// - override the wait duration to SentinelDuration (with any return values).
func Retry1[T any](ctx context.Context, pauseSeq iter.Seq[time.Duration], f func(int, *time.Duration) (T, error)) (T, error) {
	var t T
	var err error
	for i, nw := range Backoff(ctx, pauseSeq) {
		t, err = f(i, nw)
		if errors.Is(err, ErrUnrecoverable) {
			return t, err
		}
		if err != nil {
			continue
		}
		return t, nil
	}
	return t, err
}

// Retry2 retries a function if it returns a non-nil error, with pauses between
// retries taken from pauseSeq. It returns the values from the last call to f.
// The function is passed the iteration index and a pointer to the next wait
// duration. The function can end the retry loop early in two ways:
// - return ErrUnrecoverable (or an error wrapping ErrUnrecoverable), or
// - override the wait duration to SentinelDuration (with any return values).
func Retry2[T1, T2 any](ctx context.Context, pauseSeq iter.Seq[time.Duration], f func(int, *time.Duration) (T1, T2, error)) (T1, T2, error) {
	var t1 T1
	var t2 T2
	var err error
	for i, nw := range Backoff(ctx, pauseSeq) {
		t1, t2, err = f(i, nw)
		if errors.Is(err, ErrUnrecoverable) {
			return t1, t2, err
		}
		if err != nil {
			continue
		}
		return t1, t2, nil
	}
	return t1, t2, err
}

// Retry3 retries a function if it returns a non-nil error, with pauses between
// retries taken from pauseSeq. It returns the values from the last call to f.
// The function is passed the iteration index and a pointer to the next wait
// duration. The function can end the retry loop early in two ways:
// - return ErrUnrecoverable (or an error wrapping ErrUnrecoverable), or
// - override the wait duration to SentinelDuration (with any return values).
func Retry3[T1, T2, T3 any](ctx context.Context, pauseSeq iter.Seq[time.Duration], f func(int, *time.Duration) (T1, T2, T3, error)) (T1, T2, T3, error) {
	var t1 T1
	var t2 T2
	var t3 T3
	var err error
	for i, nw := range Backoff(ctx, pauseSeq) {
		t1, t2, t3, err = f(i, nw)
		if errors.Is(err, ErrUnrecoverable) {
			return t1, t2, t3, err
		}
		if err != nil {
			continue
		}
		return t1, t2, t3, nil
	}
	return t1, t2, t3, err
}
