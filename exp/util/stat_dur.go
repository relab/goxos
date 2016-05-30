package util

import (
	"math"
	"time"
)

func MeanDuration(v ...time.Duration) time.Duration {
	if len(v) == 0 {
		return 0
	}
	var sum time.Duration
	for _, dur := range v {
		sum += dur
	}
	return sum / time.Duration((len(v)))
}

func SSDDuration(v ...time.Duration) time.Duration {
	if len(v) == 0 {
		return 0
	}
	mean := MeanDuration(v...)
	var sum float64
	for _, dur := range v {
		sum += math.Pow(float64(dur.Nanoseconds()-mean.Nanoseconds()), 2)
	}
	svar := sum / float64(len(v)-1)
	return time.Duration(math.Sqrt(svar))
}

func StdErrMeanDuration(ssd time.Duration, n int) time.Duration {
	return time.Duration(float64(ssd.Nanoseconds()) / math.Sqrt(float64(n)))
}

func MaxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func MinDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func MinDurationN(v ...time.Duration) time.Duration {
	switch len(v) {
	case 0:
		return 0
	case 1:
		return v[0]
	case 2:
		return MinDuration(v[0], v[1])
	default:
		l := len(v) / 2
		return MinDurationN(MinDurationN(v[:l]...), MinDurationN(v[l:]...))
	}
}

func MaxDurationN(v ...time.Duration) time.Duration {
	switch len(v) {
	case 0:
		return 0
	case 1:
		return v[0]
	case 2:
		return MaxDuration(v[0], v[1])
	default:
		l := len(v) / 2
		return MaxDurationN(MaxDurationN(v[:l]...), MaxDurationN(v[l:]...))
	}
}
