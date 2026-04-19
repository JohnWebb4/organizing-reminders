package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const DAYS_LIMIT = 20

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: organizing-reminders <folder>")
		os.Exit(1)
	}
	folder := os.Args[1]

	matches, err := filepath.Glob(filepath.Join(folder, "*.ics"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var reminders []*Reminder
	for _, path := range matches {
		rem, err := parseICS(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
			continue
		}
		if rem.Status != "COMPLETED" {
			reminders = append(reminders, rem)
		}
	}

	fmt.Printf("Loaded %d reminders\n\n", len(reminders))

	// Build day → reminders map over the next year.
	now := time.Now().UTC()
	windowStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.AddDate(1, 0, 0)

	dayMap := make(map[string][]*Reminder)
	for _, rem := range reminders {
		for _, occ := range rem.Occurrences(windowStart, windowEnd) {
			key := occ.Format(time.DateOnly)
			dayMap[key] = append(dayMap[key], rem)
		}
	}

	// Sort days by descending occurrence count.
	type dayStat struct {
		day   string
		count int
	}
	stats := make([]dayStat, 0, len(dayMap))
	for day, rems := range dayMap {
		stats = append(stats, dayStat{day, len(rems)})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].count != stats[j].count {
			return stats[i].count > stats[j].count
		}
		return stats[i].day < stats[j].day
	})

	// Print top x busiest days.
	limit := DAYS_LIMIT
	if len(stats) < DAYS_LIMIT {
		limit = len(stats)
	}
	fmt.Printf("Top %d busiest days:\n", limit)
	for _, s := range stats[:limit] {
		fmt.Printf("  %s — %d reminders\n", s.day, s.count)
	}

	// Calculate and print distribution statistics.
	counts := make([]int, len(stats))
	for i, s := range stats {
		counts[i] = s.count
	}
	ds := calcStats(counts)
	fmt.Printf("\nDistribution statistics (%d days with reminders):\n", ds.Total)
	fmt.Printf("  Mean:   %.2f reminders/day\n", ds.Mean)
	fmt.Printf("  Median: %.2f reminders/day\n", ds.Median)
	fmt.Printf("  Mode:   %d reminders/day\n", ds.Mode)
	fmt.Printf("  StdDev: %.2f\n", ds.StdDev)
	fmt.Printf("  Within 1 StdDev (%.2f–%.2f): %d days (%.1f%%)\n",
		ds.Mean-ds.StdDev, ds.Mean+ds.StdDev,
		ds.Within1StdDev, 100*float64(ds.Within1StdDev)/float64(ds.Total))
	fmt.Printf("  Within 2 StdDev (%.2f–%.2f): %d days (%.1f%%)\n",
		ds.Mean-2*ds.StdDev, ds.Mean+2*ds.StdDev,
		ds.Within2StdDev, 100*float64(ds.Within2StdDev)/float64(ds.Total))
	fmt.Printf("  Within 3 StdDev (%.2f–%.2f): %d days (%.1f%%)\n",
		ds.Mean-3*ds.StdDev, ds.Mean+3*ds.StdDev,
		ds.Within3StdDev, 100*float64(ds.Within3StdDev)/float64(ds.Total))
}
