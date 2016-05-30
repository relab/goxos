package util

import "math"

func MeanUint64(v ...uint64) float64 {
	if len(v) == 0 {
		return 0
	}
	var sum uint64
	for _, i := range v {
		sum += i
	}
	return float64(sum) / float64(len(v))
}

func SSDUint64(v ...uint64) float64 {
	if len(v) == 0 {
		return 0
	}
	mean := MeanUint64(v...)
	var sum float64
	for _, i := range v {
		sum += math.Pow(float64(i)-mean, 2)
	}
	svar := sum / float64(len(v)-1)
	return math.Sqrt(svar)
}

func StdErrMeanFloat(ssd float64, n int) float64 {
	return ssd / math.Sqrt(float64(n))
}
