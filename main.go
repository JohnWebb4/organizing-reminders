package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

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

	fmt.Printf("Loaded %d reminders\n", len(reminders))

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

	// Print top 10.
	fmt.Println("\nTop 10 busiest days:")
	limit := 20
	if len(stats) < limit {
		limit = len(stats)
	}
	for _, s := range stats[:limit] {
		fmt.Printf("  %s — %d reminders\n", s.day, s.count)
	}
}
