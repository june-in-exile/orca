package suiauth

import "time"

// Clock provides the current Unix timestamp. Swappable in tests.
type Clock interface {
	Now() int64
}

type realClock struct{}

func (realClock) Now() int64 { return time.Now().Unix() }

// SystemClock returns a Clock backed by the system clock.
func SystemClock() Clock { return realClock{} }

type fixedClock struct{ ts int64 }

func (f fixedClock) Now() int64 { return f.ts }

// FixedClock returns a Clock that always returns the given timestamp.
func FixedClock(ts int64) Clock { return fixedClock{ts: ts} }
