package roko

import (
	"context"
	"errors"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/assert/opt"
)

// Insomniac implements a sleep function, but it doesn't actually sleep, it just notes down the intervals it was
// told to sleep
type insomniac struct {
	sleepIntervals []time.Duration
}

func newInsomniac() *insomniac {
	return &insomniac{sleepIntervals: []time.Duration{}}
}

func (i *insomniac) sleep(interval time.Duration) {
	i.sleepIntervals = append(i.sleepIntervals, interval)
}

func dummySleep(interval time.Duration) {}

func DurationExact() gocmp.Option {
	return gocmp.Comparer(func(x, y time.Duration) bool {
		return x == y
	})
}

var errDummy = errors.New("this makes it retry")

func TestDo(t *testing.T) {
	t.Parallel()

	i := newInsomniac()
	err := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 0)),
		WithMaxAttempts(5),
		WithSleepFunc(i.sleep),
	).Do(func(_ *Retrier) error {
		return errDummy
	})
	assert.ErrorIs(t, err, errDummy)

	assert.DeepEqual(t, []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		// There are only four waits, because after the fifth try (the fourth wait), the retrier gives up
	}, i.sleepIntervals, DurationExact())
}

func TestDoWithContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	retrier := NewRetrier(WithStrategy(Constant(1*time.Second)), TryForever())

	err := retrier.DoWithContext(ctx, func(*Retrier) error {
		t.Log("Should try once")
		return errDummy
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDo_OnSuccess_ReturnsNil(t *testing.T) {
	t.Parallel()

	callcount := 0
	i := newInsomniac()
	err := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 0)),
		WithMaxAttempts(50),
		WithSleepFunc(i.sleep),
	).Do(func(_ *Retrier) error {
		callcount += 1
		if callcount >= 9 {
			// It "succeeded" on the 9th try
			return nil
		}
		return errDummy
	})

	assert.NilError(t, err)
	assert.Equal(t, 9, callcount)
}

func TestShouldGiveUp_WithMaxAttempts(t *testing.T) {
	t.Parallel()

	callcount := 0

	err := NewRetrier(
		WithStrategy(Constant(1*time.Second)),
		WithMaxAttempts(3),
		WithSleepFunc(dummySleep),
	).Do(func(_ *Retrier) error {
		callcount += 1
		return errDummy
	})
	assert.ErrorIs(t, err, errDummy)

	assert.Equal(t, 3, callcount)
}

func TestShouldGiveUp_Break(t *testing.T) {
	t.Parallel()

	callcount := 0
	err := NewRetrier(
		WithStrategy(Constant(1*time.Second)),
		WithMaxAttempts(500),
		WithSleepFunc(dummySleep),
	).Do(func(r *Retrier) error {
		callcount += 1

		if callcount > 250 {
			r.Break()
		}

		return errDummy
	})
	assert.ErrorIs(t, err, errDummy)

	assert.Equal(t, 251, callcount)
}

func TestShouldGiveUp_Forever(t *testing.T) {
	t.Parallel()

	err := NewRetrier(
		WithStrategy(Constant(1*time.Second)),
		TryForever(),
		WithSleepFunc(dummySleep),
	).Do(func(r *Retrier) error {
		assert.Check(t, r.ShouldGiveUp() == false)

		if r.AttemptCount() == 250_000 { // an arbitrarily large number of retries
			return nil
		}

		return errDummy
	})
	assert.NilError(t, err)
}

func TestNextInterval_ConstantStrategy(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	err := NewRetrier(
		WithStrategy(Constant(5*time.Second)),
		WithMaxAttempts(1000),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })
	assert.ErrorIs(t, err, errDummy)

	for _, interval := range insomniac.sleepIntervals {
		assert.Check(t, interval == 5*time.Second)
	}
}

func TestNextInterval_ConstantStrategy_WithJitter(t *testing.T) {
	t.Parallel()

	expected := 5 * time.Second
	insomniac := newInsomniac()

	err := NewRetrier(
		WithStrategy(Constant(expected)),
		WithJitter(),
		WithMaxAttempts(1000),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })
	assert.ErrorIs(t, err, errDummy)

	for _, interval := range insomniac.sleepIntervals {
		assert.Check(t, cmp.DeepEqual(interval, expected, opt.DurationWithThreshold(jitterInterval)))
	}
}

func TestNextInterval_ExponentialStrategy(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()

	err := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 0)),
		WithMaxAttempts(5),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })
	assert.ErrorIs(t, err, errDummy)

	assert.DeepEqual(t,
		[]time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
			8 * time.Second,
		},
		insomniac.sleepIntervals,
		DurationExact(),
	)
}

func TestNextInterval_ExponentialStrategy_WithAdjustment(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	err := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 3*time.Second)),
		WithMaxAttempts(6),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })

	assert.ErrorIs(t, err, errDummy)

	assert.Assert(t,
		cmp.DeepEqual(
			[]time.Duration{
				4 * time.Second,
				5 * time.Second,
				7 * time.Second,
				11 * time.Second,
				19 * time.Second,
				// There are only four waits, because after the fifth try (the fourth wait), the retrier gives up
			},
			insomniac.sleepIntervals,
			DurationExact(),
		),
	)
}

func TestNextInterval_ExponentialStrategy_WithJitter(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	err := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 0)),
		WithJitter(),
		WithMaxAttempts(6),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })
	assert.ErrorIs(t, err, errDummy)

	assert.DeepEqual(t,
		[]time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
			8 * time.Second,
			16 * time.Second,
		},
		insomniac.sleepIntervals,
		opt.DurationWithThreshold(jitterInterval),
	)
}

func TestNextInterval_ExponentialSubsecondStrategy_100ms(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	err := NewRetrier(
		WithStrategy(ExponentialSubsecond(100*time.Millisecond)),
		WithMaxAttempts(10),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })

	assert.Error(t, err)

	// very short initial delay of 100ms grows to quite short 1 second for 10 attempts
	assert.Equal(t, []time.Duration{
		100 * time.Millisecond,
		133 * time.Millisecond,
		177 * time.Millisecond,
		237 * time.Millisecond,
		316 * time.Millisecond,
		421 * time.Millisecond,
		562 * time.Millisecond,
		749 * time.Millisecond,
		1000 * time.Millisecond,
	}, insomniac.sleepIntervals)
}

func TestNextInterval_ExponentialSubsecondStrategy_1sec(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	err := NewRetrier(
		WithStrategy(ExponentialSubsecond(1*time.Second)),
		WithMaxAttempts(10),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })

	assert.Error(t, err)

	// reasonable initial delay of 1 second grows to reasonable 30 seconds for 10 attempts
	assert.Equal(t, []time.Duration{
		1000 * time.Millisecond,
		1539 * time.Millisecond,
		2371 * time.Millisecond,
		3651 * time.Millisecond,
		5623 * time.Millisecond,
		8659 * time.Millisecond,
		13335 * time.Millisecond,
		20535 * time.Millisecond,
		31622 * time.Millisecond,
	}, insomniac.sleepIntervals)
}

func TestNextInterval_ExponentialSubsecondStrategy_5sec(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	err := NewRetrier(
		WithStrategy(ExponentialSubsecond(5*time.Second)),
		WithMaxAttempts(10),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })

	assert.Error(t, err)

	// quite long 5 second initial delay grows to long-but-reasonable ~6 minutes for 10 attempts
	assert.Equal(t, []time.Duration{
		5000 * time.Millisecond,
		8514 * time.Millisecond,
		14499 * time.Millisecond,
		24690 * time.Millisecond,
		42044 * time.Millisecond,
		71597 * time.Millisecond,
		121922 * time.Millisecond,
		207620 * time.Millisecond,
		353553 * time.Millisecond,
	}, insomniac.sleepIntervals)
}

func TestNextInterval_ExponentialSubsecondStrategy_WithJitter(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	err := NewRetrier(
		WithStrategy(ExponentialSubsecond(1*time.Second)),
		WithJitter(),
		WithMaxAttempts(10),
		WithSleepFunc(insomniac.sleep),
	).Do(func(_ *Retrier) error { return errDummy })

	assert.Error(t, err)

	expectedIntervals := []time.Duration{
		1000 * time.Millisecond,
		1539 * time.Millisecond,
		2371 * time.Millisecond,
		3651 * time.Millisecond,
		5623 * time.Millisecond,
		8659 * time.Millisecond,
		13335 * time.Millisecond,
		20535 * time.Millisecond,
		31622 * time.Millisecond,
	}

	for idx, actualInterval := range insomniac.sleepIntervals {
		assert.Truef(
			t,
			withinJitterInterval(actualInterval, expectedIntervals[idx]),
			"actual interval %v wasn't within 1s of expected interval %v", actualInterval, expectedIntervals[idx],
		)
	}
}

func TestString_WithFiniteAttemptCount(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	r := NewRetrier(
		WithStrategy(Constant(1*time.Second)),
		WithMaxAttempts(5),
		WithSleepFunc(insomniac.sleep),
	)

	retryingIns := make([]string, 0, 5)
	err := r.Do(func(_ *Retrier) error {
		retryingIns = append(retryingIns, r.String())
		return errDummy
	})
	assert.ErrorIs(t, err, errDummy)

	assert.DeepEqual(t,
		[]string{
			"Attempt 1/5 Retrying in 1s",
			"Attempt 2/5 Retrying in 1s",
			"Attempt 3/5 Retrying in 1s",
			"Attempt 4/5 Retrying in 1s",
			"Attempt 5/5",
		},
		retryingIns,
	)
}

func TestString_WithExponentialStrategy(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	r := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 0)),
		WithMaxAttempts(5),
		WithSleepFunc(insomniac.sleep),
	)

	retryingIns := make([]string, 0, 5)
	err := r.Do(func(_ *Retrier) error {
		retryingIns = append(retryingIns, r.String())
		return errDummy
	})
	assert.ErrorIs(t, err, errDummy)

	assert.DeepEqual(t,
		[]string{
			"Attempt 1/5 Retrying in 1s",
			"Attempt 2/5 Retrying in 2s",
			"Attempt 3/5 Retrying in 4s",
			"Attempt 4/5 Retrying in 8s",
			"Attempt 5/5",
		},
		retryingIns,
	)
}

func TestString_WithTryForever(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()
	r := NewRetrier(
		WithStrategy(Constant(1*time.Second)),
		TryForever(),
		WithSleepFunc(insomniac.sleep),
	)

	retryingIns := make([]string, 0, 5)
	err := r.Do(func(_ *Retrier) error {
		if r.AttemptCount() >= 5 {
			r.Break()
			return nil
		}

		retryingIns = append(retryingIns, r.String())

		return errDummy
	})
	assert.NilError(t, err)

	assert.DeepEqual(t, []string{
		"Attempt 1/∞ Retrying in 1s",
		"Attempt 2/∞ Retrying in 1s",
		"Attempt 3/∞ Retrying in 1s",
		"Attempt 4/∞ Retrying in 1s",
		"Attempt 5/∞ Retrying in 1s",
	}, retryingIns)
}

func TestString_WithNoDelay(t *testing.T) {
	t.Parallel()

	r := NewRetrier(
		WithStrategy(Constant(0)),
		WithMaxAttempts(5),
	)

	retryingIns := make([]string, 0, 5)
	err := r.Do(func(_ *Retrier) error {
		if r.AttemptCount() >= 5 {
			r.Break()
			return nil
		}

		retryingIns = append(retryingIns, r.String())

		return errDummy
	})
	assert.ErrorIs(t, err, errDummy)

	assert.DeepEqual(t, []string{
		"Attempt 1/5 Retrying immediately",
		"Attempt 2/5 Retrying immediately",
		"Attempt 3/5 Retrying immediately",
		"Attempt 4/5 Retrying immediately",
		"Attempt 5/5",
	}, retryingIns)
}

func TestSetNextInterval_Strings(t *testing.T) {
	t.Parallel()

	strings := []string{}

	err := NewRetrier(
		WithStrategy(Constant(10*time.Second)),
		WithMaxAttempts(5),
		WithSleepFunc(dummySleep),
	).Do(func(r *Retrier) error {
		switch r.AttemptCount() {
		case 1:
			r.SetNextInterval(0 * time.Second)
		case 3:
			r.SetNextInterval(4 * time.Second)
		}
		strings = append(strings, r.String())
		return errDummy
	})
	assert.ErrorIs(t, err, errDummy)

	assert.DeepEqual(t, []string{
		"Attempt 1/5 Retrying in 10s", // default
		"Attempt 2/5 Retrying immediately",
		"Attempt 3/5 Retrying in 10s", // default
		"Attempt 4/5 Retrying in 4s",
		"Attempt 5/5",
	}, strings)
}

func TestSetNextInterval_Interval(t *testing.T) {
	t.Parallel()

	insomniac := newInsomniac()

	err := NewRetrier(
		WithStrategy(Constant(2*time.Second)),
		WithMaxAttempts(5),
		WithSleepFunc(insomniac.sleep),
	).Do(func(r *Retrier) error {
		switch r.AttemptCount() {
		case 1:
			r.SetNextInterval(0 * time.Second)
		case 3:
			r.SetNextInterval(4 * time.Second)
		}
		return errDummy
	})
	assert.ErrorIs(t, err, errDummy)

	assert.DeepEqual(t, []time.Duration{
		2 * time.Second, // default
		0 * time.Second, // manual
		2 * time.Second, // default
		4 * time.Second, // manual
	}, insomniac.sleepIntervals, DurationExact())
}
