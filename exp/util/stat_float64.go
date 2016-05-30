package util

import (
	"sort"
)

func MeanFloat64(v ...float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var sum float64
	for _, i := range v {
		sum += i
	}
	return sum / float64(len(v))
}

func MeanFloat64Map(Map map[string]float64) float64 {
	if len(Map) == 0 {
		return 0
	}
	var sum float64
	for _, i := range Map {
		sum += i
	}
	return sum / float64(len(Map))
}

func MedianFloat64Map(Map map[string]float64) string {
	if len(Map) == 0 {
		return ""
	}
	slice := make([]float64, 0, len(Map))
	for _, lat := range Map {
		slice = append(slice, lat)
	}
	median := MedianFloatSlice(slice)
	for k, l := range Map {
		if l == median {
			return k
		}
	}
	return ""

}

func MaxFloat64Map(Map map[string]float64) string {
	if len(Map) == 0 {
		return ""
	}
	max := float64(0)
	cl := ""
	for k, lat := range Map {
		if lat > max {
			max = lat
			cl = k
		}
	}
	return cl
}

func MedianFloatSlice(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sort.Float64s(v)
	return v[len(v)/2]
}

func MedianMinMax90FloatSlice(v []float64) (median, min90, max90 float64) {
	if len(v) == 0 {
		return 0, 0, 0
	}
	sort.Float64s(v)
	return v[len(v)/2], v[len(v)/10], v[(len(v)*9)/10]
}
