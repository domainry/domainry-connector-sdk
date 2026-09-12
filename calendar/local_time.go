package calendar

import (
	"fmt"
	"time"
)

// LocalDateTime resolves a source wall-clock value only when the IANA zone
// gives a unique instant. Missing and repeated DST times require an explicit
// offset from the source; Go's implicit choice must not become calendar evidence.
func LocalDateTime(value, zone string) (string, error) {
	if err := validateZone(zone); err != nil {
		return "", err
	}
	wall, err := time.Parse("2006-01-02T15:04:05", value)
	if err != nil {
		return "", fmt.Errorf("invalid local calendar date_time")
	}
	loc, _ := time.LoadLocation(zone)
	approx := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), loc)
	offsets := map[int]bool{}
	add := func(t time.Time) { _, offset := t.In(loc).Zone(); offsets[offset] = true }
	add(approx)
	add(approx.Add(-48 * time.Hour))
	add(approx.Add(48 * time.Hour))
	begin, end := approx.ZoneBounds()
	if !begin.IsZero() {
		add(begin.Add(-time.Second))
	}
	if !end.IsZero() {
		add(end)
	}
	var matches []time.Time
	for offset := range offsets {
		candidate := wall.Add(-time.Duration(offset) * time.Second).In(loc)
		if candidate.Format("2006-01-02T15:04:05.999999999") == wall.Format("2006-01-02T15:04:05.999999999") {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("calendar local time is missing or ambiguous; source offset required")
	}
	return matches[0].Format(time.RFC3339Nano), nil
}
