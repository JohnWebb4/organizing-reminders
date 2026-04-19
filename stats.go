package main

import (
	"math"
	"sort"
)

type DayStats struct {
	Mean   float64
	Median float64
	Mode   int
	StdDev float64
	Within1StdDev int
	Within2StdDev int
	Within3StdDev int
	Total  int
}

func calcStats(counts []int) DayStats {
	n := len(counts)
	if n == 0 {
		return DayStats{}
	}

	sorted := make([]int, n)
	copy(sorted, counts)
	sort.Ints(sorted)

	// Mean
	sum := 0
	for _, c := range sorted {
		sum += c
	}
	mean := float64(sum) / float64(n)

	// Median
	var median float64
	if n%2 == 0 {
		median = float64(sorted[n/2-1]+sorted[n/2]) / 2.0
	} else {
		median = float64(sorted[n/2])
	}

	// Mode
	freq := make(map[int]int)
	for _, c := range sorted {
		freq[c]++
	}
	mode, maxFreq := 0, 0
	for val, f := range freq {
		if f > maxFreq || (f == maxFreq && val > mode) {
			mode = val
			maxFreq = f
		}
	}

	// Standard deviation (population)
	variance := 0.0
	for _, c := range sorted {
		diff := float64(c) - mean
		variance += diff * diff
	}
	variance /= float64(n)
	stdDev := math.Sqrt(variance)

	// Days within 1, 2, 3 standard deviations
	within1, within2, within3 := 0, 0, 0
	for _, c := range sorted {
		dist := math.Abs(float64(c) - mean)
		if dist <= stdDev {
			within1++
		}
		if dist <= 2*stdDev {
			within2++
		}
		if dist <= 3*stdDev {
			within3++
		}
	}

	return DayStats{
		Mean:          mean,
		Median:        median,
		Mode:          mode,
		StdDev:        stdDev,
		Within1StdDev: within1,
		Within2StdDev: within2,
		Within3StdDev: within3,
		Total:         n,
	}
}
