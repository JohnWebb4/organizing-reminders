package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// RRule holds parsed recurrence rule fields from an RRULE line.
type RRule struct {
	Freq     string   // YEARLY, MONTHLY, WEEKLY, DAILY
	ByDay    []string // e.g. ["SU", "WE"]
	Interval int      // default 1
	BySetPos int      // e.g. 1 for "first"
}

// Reminder represents a single parsed VTODO from an ICS file.
type Reminder struct {
	UID     string
	Status  string
	Summary string
	DtStart string
	RRule   *RRule
}

// Occurrences returns all dates this reminder falls on in [windowStart, windowEnd].
func (r *Reminder) Occurrences(windowStart time.Time, windowEnd time.Time) []time.Time {
	dtStart, err := parseDtStart(r.DtStart)
	if err != nil {
		return nil
	}

	// Normalize to midnight so comparisons are day-based.
	dtStart = time.Date(dtStart.Year(), dtStart.Month(), dtStart.Day(), 0, 0, 0, 0, time.UTC)

	if r.RRule == nil {
		if !dtStart.Before(windowStart) && !dtStart.After(windowEnd) {
			return []time.Time{dtStart}
		}
		return nil
	}

	return findOccurrences(dtStart, r.RRule, windowStart, windowEnd)
}

func (r *Reminder) String() string {
	if r.RRule != nil {
		return fmt.Sprintf("%s (starts %s, repeats %s every %d, byday=%v, bysetpos=%d)", r.Summary, r.DtStart, r.RRule.Freq, r.RRule.Interval, r.RRule.ByDay, r.RRule.BySetPos)
	}
	return fmt.Sprintf("%s (starts %s)", r.Summary, r.DtStart)
}

// parseICS parses a single ICS file and returns a Reminder.
func parseICS(path string) (*Reminder, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rem := &Reminder{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		// DTSTART may have parameters: "DTSTART;VALUE=DATE:20260720"
		baseKey := strings.SplitN(key, ";", 2)[0]
		switch baseKey {
		case "UID":
			rem.UID = value
		case "SUMMARY":
			rem.Summary = value
		case "STATUS":
			rem.Status = value
		case "DTSTART":
			rem.DtStart = value
		case "RRULE":
			rem.RRule, err = parseRRule(value)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rem, nil
}

// parseRRule parses the value portion of an RRULE line (everything after "RRULE:").
// Example inputs:
//
//	FREQ=YEARLY
//	FREQ=WEEKLY;BYDAY=SU,WE
//	FREQ=MONTHLY;INTERVAL=6;BYSETPOS=1;BYDAY=SU,SA
func parseRRule(value string) (*RRule, error) {
	r := &RRule{Interval: 1}
	for _, part := range strings.Split(value, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]
		switch key {
		case "FREQ":
			r.Freq = val
		case "BYDAY":
			r.ByDay = strings.Split(val, ",")
		case "INTERVAL":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid INTERVAL %q: %w", val, err)
			}
			r.Interval = n
		case "BYSETPOS":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid BYSETPOS %q: %w", val, err)
			}
			r.BySetPos = n
		}
	}
	if r.Freq == "" {
		return nil, fmt.Errorf("RRULE missing FREQ")
	}
	return r, nil
}

// parseDtStart parses an ICS DTSTART value into a time.Time (date-only or datetime).
func parseDtStart(s string) (time.Time, error) {
	for _, layout := range []string{"20060102T150405Z", "20060102T150405", "20060102"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse DTSTART %q", s)
}
