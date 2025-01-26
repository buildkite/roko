package roko

import (
	"time"

	gocmp "github.com/google/go-cmp/cmp"
)

func DurationExact() gocmp.Option {
	return gocmp.Comparer(func(x, y time.Duration) bool {
		return x == y
	})
}
