package main

import (
	"container/heap"
	"fmt"
	"hash/fnv"
	"strings"
	"time"
)

const windowDays = 366

// ShiftType represents how a reminder can be rescheduled while keeping the same frequency.
type ShiftType int

const (
	ShiftNone        ShiftType = iota
	ShiftWeekday               // WEEKLY: rotate which day(s) of the week [0,6]
	ShiftDayOfMonth            // MONTHLY (no BYDAY): move to a different day of the month
	ShiftWeekOfMonth           // MONTHLY (BYDAY+BYSETPOS): move to a different week of the month
	ShiftDayOfYear             // YEARLY: move to a different day of the year
)

// reminderInfo holds the classification for A* search.
type reminderInfo struct {
	reminder  *Reminder
	dtStart   time.Time
	shiftType ShiftType
	minOffset int
	maxOffset int
}

// OptimizeSuggestion describes a recommended schedule change for one reminder.
type OptimizeSuggestion struct {
	Reminder *Reminder
	Offset   int
	From     string
	To       string
}

// aStarState is a node in the A* search tree.
type aStarState struct {
	offsets []int
	counts  [windowDays]uint8 // occurrences per day (index 0 = windowStart)
	gCost   int               // actions taken
	hCost   int               // estimated actions remaining
	heapIdx int
}

func (s *aStarState) fCost() int { return s.gCost + s.hCost }

type aStarHeap []*aStarState

func (h aStarHeap) Len() int           { return len(h) }
func (h aStarHeap) Less(i, j int) bool { return h[i].fCost() < h[j].fCost() }
func (h aStarHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIdx = i
	h[j].heapIdx = j
}
func (h *aStarHeap) Push(x interface{}) {
	s := x.(*aStarState)
	s.heapIdx = len(*h)
	*h = append(*h, s)
}
func (h *aStarHeap) Pop() interface{} {
	old := *h
	n := len(old)
	s := old[n-1]
	old[n-1] = nil
	s.heapIdx = -1
	*h = old[:n-1]
	return s
}

// classifyReminder determines the shift type and valid offset range for one reminder.
func classifyReminder(r *Reminder) reminderInfo {
	info := reminderInfo{reminder: r, shiftType: ShiftNone}
	if r.RRule == nil {
		return info
	}
	dt, err := parseDtStart(r.DtStart)
	if err != nil {
		return info
	}
	info.dtStart = dt

	switch RRuleFrequency(r.RRule.Freq) {
	case FREQ_WEEKLY:
		info.shiftType = ShiftWeekday
		info.minOffset = 0
		info.maxOffset = 6

	case FREQ_MONTHLY:
		if len(r.RRule.ByDay) > 0 && r.RRule.BySetPos > 0 {
			// e.g. BYSETPOS=1;BYDAY=TU → "1st Tuesday". Shift to a different week.
			pos := r.RRule.BySetPos
			info.shiftType = ShiftWeekOfMonth
			info.minOffset = 1 - pos
			info.maxOffset = 4 - pos
		} else if len(r.RRule.ByDay) == 0 {
			// Plain day-of-month (e.g. the 15th). Limit to days 1–28 (safe for all months).
			day := dt.Day()
			info.shiftType = ShiftDayOfMonth
			info.minOffset = 1 - day
			info.maxOffset = 28 - day
			if info.maxOffset < 0 {
				info.maxOffset = 0 // day > 28: can only shift down
			}
		}

	case FREQ_YEARLY:
		info.shiftType = ShiftDayOfYear
		info.minOffset = -182
		info.maxOffset = 182
	}
	return info
}

// applyOffset returns a shallow-copied Reminder with the shift applied.
func applyOffset(info reminderInfo, offset int) *Reminder {
	if offset == 0 {
		return info.reminder
	}
	r := *info.reminder
	if info.reminder.RRule != nil {
		rr := *info.reminder.RRule
		r.RRule = &rr
	}
	switch info.shiftType {
	case ShiftWeekday:
		r.DtStart = fmtDtStart(info.dtStart.AddDate(0, 0, offset), info.reminder.DtStart)
		if len(info.reminder.RRule.ByDay) > 0 {
			r.RRule.ByDay = shiftByDay(info.reminder.RRule.ByDay, offset)
		}
	case ShiftDayOfMonth:
		newDay := info.dtStart.Day() + offset
		newDate := time.Date(info.dtStart.Year(), info.dtStart.Month(), newDay, 0, 0, 0, 0, time.UTC)
		r.DtStart = fmtDtStart(newDate, info.reminder.DtStart)
	case ShiftWeekOfMonth:
		pos := info.reminder.RRule.BySetPos + offset
		if pos < 1 {
			pos = 1
		} else if pos > 4 {
			pos = 4
		}
		r.RRule.BySetPos = pos
	case ShiftDayOfYear:
		r.DtStart = fmtDtStart(info.dtStart.AddDate(0, 0, offset), info.reminder.DtStart)
	}
	return &r
}

// shiftByDay rotates every entry in a BYDAY list by n weekdays.
func shiftByDay(byDay []string, n int) []string {
	order := [7]string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}
	pos := map[string]int{"SU": 0, "MO": 1, "TU": 2, "WE": 3, "TH": 4, "FR": 5, "SA": 6}
	result := make([]string, len(byDay))
	for i, d := range byDay {
		result[i] = order[((pos[d]+n)%7+7)%7]
	}
	return result
}

// fmtDtStart re-encodes t using the same format as the original DtStart value string.
func fmtDtStart(t time.Time, original string) string {
	switch len(original) {
	case 8: // "20060102"
		return t.Format(string(DTStartLayoutDate))
	case 16: // "20060102T150405Z"
		return t.UTC().Format(string(DTStartLayoutDateTimeUTC))
	default: // "20060102T150405"
		return t.Format(string(DTStartLayoutDateTime))
	}
}

// occCache caches occurrence day-indices per shift offset for one reminder.
type occCache struct {
	info        reminderInfo
	windowStart time.Time
	windowEnd   time.Time
	data        map[int][]int // offset → sorted list of day indices into counts array
}

func newOccCache(info reminderInfo, ws, we time.Time) *occCache {
	return &occCache{info: info, windowStart: ws, windowEnd: we, data: make(map[int][]int)}
}

func (c *occCache) get(offset int) []int {
	if c.info.shiftType == ShiftWeekday {
		offset = ((offset % 7) + 7) % 7 // normalize to [0,6]
	}
	if cached, ok := c.data[offset]; ok {
		return cached
	}
	r := applyOffset(c.info, offset)
	occs := r.Occurrences(c.windowStart, c.windowEnd)
	indices := make([]int, 0, len(occs))
	for _, occ := range occs {
		idx := int(occ.Sub(c.windowStart).Hours() / 24)
		if idx >= 0 && idx < windowDays {
			indices = append(indices, idx)
		}
	}
	c.data[offset] = indices
	return indices
}

// hashCounts returns a fast 64-bit hash of the day-count array for state deduplication.
func hashCounts(counts [windowDays]uint8) uint64 {
	h := fnv.New64a()
	h.Write(counts[:])
	return h.Sum64()
}

// calcH returns the number of days above mean+stddev — the A* heuristic.
func calcH(counts [windowDays]uint8) int {
	vals := make([]int, 0, windowDays)
	for _, c := range counts {
		if c > 0 {
			vals = append(vals, int(c))
		}
	}
	if len(vals) == 0 {
		return 0
	}
	ds := calcStats(vals)
	threshold := ds.Mean + ds.StdDev
	n := 0
	for _, c := range vals {
		if float64(c) > threshold {
			n++
		}
	}
	return n
}

// Optimize runs A* to find reminder shifts that minimize days above mean+stddev.
//
// The search state is the day-count distribution. Two offset vectors that produce
// the same distribution are treated as the same state (stateless deduplication).
// g(n) = actions taken; h(n) = days outside 1 standard deviation.
//
// Returns the suggested changes and the optimized day-count array.
func Optimize(reminders []*Reminder, windowStart, windowEnd time.Time) ([]OptimizeSuggestion, [windowDays]uint8) {
	infos := make([]reminderInfo, len(reminders))
	for i, r := range reminders {
		infos[i] = classifyReminder(r)
	}

	caches := make([]*occCache, len(reminders))
	for i, info := range infos {
		caches[i] = newOccCache(info, windowStart, windowEnd)
	}

	// Build the initial counts array from offset-0 occurrences.
	var initialCounts [windowDays]uint8
	for i := range reminders {
		for _, idx := range caches[i].get(0) {
			initialCounts[idx]++
		}
	}

	initialH := calcH(initialCounts)
	if initialH == 0 {
		return nil, initialCounts
	}

	initial := &aStarState{
		offsets: make([]int, len(reminders)),
		counts:  initialCounts,
		gCost:   0,
		hCost:   initialH,
	}

	pq := &aStarHeap{initial}
	heap.Init(pq)

	visited := map[uint64]int{hashCounts(initialCounts): 0}
	best := initial

	const maxIterations = 10000
	const maxActions = 20

	for iter := 0; pq.Len() > 0 && iter < maxIterations; iter++ {
		cur := heap.Pop(pq).(*aStarState)

		if cur.hCost < best.hCost || (cur.hCost == best.hCost && cur.gCost < best.gCost) {
			best = cur
		}
		if cur.hCost == 0 {
			break
		}
		if cur.gCost >= maxActions {
			continue
		}

		for i, info := range infos {
			if info.shiftType == ShiftNone {
				continue
			}
			curOccs := caches[i].get(cur.offsets[i])

			for _, delta := range []int{-1, +1} {
				var newOffset int
				if info.shiftType == ShiftWeekday {
					newOffset = ((cur.offsets[i] + delta) % 7 + 7) % 7
				} else {
					newOffset = cur.offsets[i] + delta
					if newOffset < info.minOffset || newOffset > info.maxOffset {
						continue
					}
				}

				newOccs := caches[i].get(newOffset)

				// Incremental update: remove old occurrences, add new ones.
				newCounts := cur.counts
				for _, idx := range curOccs {
					if newCounts[idx] > 0 {
						newCounts[idx]--
					}
				}
				for _, idx := range newOccs {
					if newCounts[idx] < 255 {
						newCounts[idx]++
					}
				}

				newG := cur.gCost + 1
				key := hashCounts(newCounts)
				if existingG, seen := visited[key]; seen && existingG <= newG {
					continue
				}
				visited[key] = newG

				newOffsets := make([]int, len(cur.offsets))
				copy(newOffsets, cur.offsets)
				newOffsets[i] = newOffset

				heap.Push(pq, &aStarState{
					offsets: newOffsets,
					counts:  newCounts,
					gCost:   newG,
					hCost:   calcH(newCounts),
				})
			}
		}
	}

	var suggestions []OptimizeSuggestion
	for i, info := range infos {
		if best.offsets[i] == 0 || info.shiftType == ShiftNone {
			continue
		}
		suggestions = append(suggestions, OptimizeSuggestion{
			Reminder: reminders[i],
			Offset:   best.offsets[i],
			From:     describePos(info, 0),
			To:       describePos(info, best.offsets[i]),
		})
	}
	return suggestions, best.counts
}

// describePos returns a human-readable description of a reminder's position at offset.
func describePos(info reminderInfo, offset int) string {
	switch info.shiftType {
	case ShiftWeekday:
		if len(info.reminder.RRule.ByDay) > 0 {
			return strings.Join(shiftByDay(info.reminder.RRule.ByDay, offset), ",")
		}
		wd := time.Weekday(((int(info.dtStart.Weekday()) + offset) % 7 + 7) % 7)
		return wd.String()
	case ShiftDayOfMonth:
		return ordinalDay(info.dtStart.Day() + offset)
	case ShiftWeekOfMonth:
		pos := info.reminder.RRule.BySetPos + offset
		if pos < 1 {
			pos = 1
		} else if pos > 4 {
			pos = 4
		}
		return fmt.Sprintf("%s %s", ordinalDay(pos), strings.Join(info.reminder.RRule.ByDay, "/"))
	case ShiftDayOfYear:
		return info.dtStart.AddDate(0, 0, offset).Format("Jan 2")
	}
	return ""
}

func ordinalDay(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		if n%100 != 11 {
			suffix = "st"
		}
	case 2:
		if n%100 != 12 {
			suffix = "nd"
		}
	case 3:
		if n%100 != 13 {
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
