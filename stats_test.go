package main

import (
	"math"
	"testing"
)

func TestCalcStats_Empty(t *testing.T) {
	ds := calcStats(nil)
	if ds.Total != 0 {
		t.Errorf("expected Total=0, got %d", ds.Total)
	}
}

func TestCalcStats(t *testing.T) {
	// counts: [1, 2, 3, 4, 5]
	// mean=3, median=3, mode=any (all freq 1, picks highest=5), stddev=sqrt(2)≈1.414
	counts := []int{1, 2, 3, 4, 5}
	ds := calcStats(counts)

	if ds.Total != 5 {
		t.Errorf("Total: got %d, want 5", ds.Total)
	}
	if ds.Mean != 3.0 {
		t.Errorf("Mean: got %f, want 3.0", ds.Mean)
	}
	if ds.Median != 3.0 {
		t.Errorf("Median: got %f, want 3.0", ds.Median)
	}
	wantStdDev := math.Sqrt(2)
	if math.Abs(ds.StdDev-wantStdDev) > 1e-9 {
		t.Errorf("StdDev: got %f, want %f", ds.StdDev, wantStdDev)
	}
	// Within 1 stddev of mean=3: range [3-1.414, 3+1.414] = [1.586, 4.414] → values 2,3,4
	if ds.Within1StdDev != 3 {
		t.Errorf("Within1StdDev: got %d, want 3", ds.Within1StdDev)
	}
	// Within 2 stddev: [0.172, 5.828] → values 1,2,3,4,5
	if ds.Within2StdDev != 5 {
		t.Errorf("Within2StdDev: got %d, want 5", ds.Within2StdDev)
	}
	if ds.Within3StdDev != 5 {
		t.Errorf("Within3StdDev: got %d, want 5", ds.Within3StdDev)
	}
}

func TestCalcStats_EvenCount(t *testing.T) {
	// counts: [1, 2, 3, 4] → median = (2+3)/2 = 2.5
	ds := calcStats([]int{1, 2, 3, 4})
	if ds.Median != 2.5 {
		t.Errorf("Median: got %f, want 2.5", ds.Median)
	}
}

func TestCalcStats_Mode(t *testing.T) {
	// counts: [1, 2, 2, 3] → mode = 2
	ds := calcStats([]int{1, 2, 2, 3})
	if ds.Mode != 2 {
		t.Errorf("Mode: got %d, want 2", ds.Mode)
	}
}
