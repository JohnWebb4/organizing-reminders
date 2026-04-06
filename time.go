package main

import (
	"fmt"
	"time"
)

type RRuleFrequency string

const (
	FREQ_DAILY   RRuleFrequency = "DAILY"
	FREQ_WEEKLY  RRuleFrequency = "WEEKLY"
	FREQ_MONTHLY RRuleFrequency = "MONTHLY"
	FREQ_YEARLY  RRuleFrequency = "YEARLY"
)

var weekdays = map[string]time.Weekday{
	"SU": time.Sunday,
	"MO": time.Monday,
	"TU": time.Tuesday,
	"WE": time.Wednesday,
	"TH": time.Thursday,
	"FR": time.Friday,
	"SA": time.Saturday,
}

// nthWeekdayInMonth returns the date of the bySetPos-th occurrence of any weekday
// in byDay within the given month (1-based positive or negative index).
func nthWeekdayInMonth(year int, month time.Month, byDay []string, bySetPos int) time.Time {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	var occurrences []time.Time
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		for _, day := range byDay {
			if wd, ok := weekdays[day]; ok && d.Weekday() == wd {
				occurrences = append(occurrences, d)
				break
			}
		}
	}
	if bySetPos > 0 && bySetPos <= len(occurrences) {
		return occurrences[bySetPos-1]
	}
	if bySetPos < 0 && -bySetPos <= len(occurrences) {
		return occurrences[len(occurrences)+bySetPos]
	}
	return time.Time{}
}

func findOccurrences(dtStart time.Time, rr *RRule, windowStart time.Time, windowEnd time.Time) []time.Time {
	var results []time.Time

	switch RRuleFrequency(rr.Freq) {
	case FREQ_DAILY:
		for t := dtStart; !t.After(windowEnd); t = t.AddDate(0, 0, rr.Interval) {
			if !t.Before(windowStart) {
				results = append(results, t)
			}
		}

	case FREQ_WEEKLY:
		if len(rr.ByDay) == 0 {
			for t := dtStart; !t.After(windowEnd); t = t.AddDate(0, 0, 7*rr.Interval) {
				if !t.Before(windowStart) {
					results = append(results, t)
				}
			}
		} else {
			// Walk week-by-week (Sunday-anchored) starting from dtStart's week.
			weekSun := dtStart.AddDate(0, 0, -int(dtStart.Weekday()))
			for ; !weekSun.After(windowEnd); weekSun = weekSun.AddDate(0, 0, 7*rr.Interval) {
				for _, day := range rr.ByDay {
					wd, ok := weekdays[day]
					if !ok {
						fmt.Printf("Failed to find weekday %v\n", day)
						continue
					}
					t := weekSun.AddDate(0, 0, int(wd))
					if !t.Before(dtStart) && !t.Before(windowStart) && !t.After(windowEnd) {
						results = append(results, t)
					}
				}
			}
		}

	case FREQ_MONTHLY:
		year, month, _ := dtStart.Date()
		if len(rr.ByDay) > 0 && rr.BySetPos != 0 {
			for m := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC); !m.After(windowEnd); m = m.AddDate(0, rr.Interval, 0) {
				t := nthWeekdayInMonth(m.Year(), m.Month(), rr.ByDay, rr.BySetPos)
				if !t.IsZero() && !t.Before(dtStart) && !t.Before(windowStart) && !t.After(windowEnd) {
					results = append(results, t)
				}
			}
		} else {
			_, _, dom := dtStart.Date()
			for t := time.Date(year, month, dom, 0, 0, 0, 0, time.UTC); !t.After(windowEnd); t = t.AddDate(0, rr.Interval, 0) {
				if !t.Before(windowStart) {
					results = append(results, t)
				}
			}
		}

	case FREQ_YEARLY:
		for t := dtStart; !t.After(windowEnd); t = t.AddDate(rr.Interval, 0, 0) {
			if !t.Before(windowStart) {
				results = append(results, t)
			}
		}

	default:
		fmt.Printf("Error: Unknown frequency: %v\n", rr.Freq)
	}

	return results
}
