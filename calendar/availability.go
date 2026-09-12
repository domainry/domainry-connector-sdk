package calendar

import (
	"fmt"
	"sort"
	"time"
)

// ResolveAvailability intersects available time for precisely the requested
// calendars. It clips and merges overlaps; an omitted calendar, partial result
// or provider error suppresses all Free intervals instead of inventing free time.
func ResolveAvailability(request AvailabilityRequest, calendars []CalendarBusy) (Availability, error) {
	if err := request.Validate(); err != nil {
		return Availability{}, err
	}
	out := Availability{Window: request.Window, Calendars: []CalendarBusy{}, Free: []Window{}, Complete: true}
	byID := map[string]CalendarBusy{}
	for _, c := range calendars {
		if _, exists := byID[c.CalendarID]; exists {
			return Availability{}, fmt.Errorf("duplicate calendar availability")
		}
		byID[c.CalendarID] = c
	}
	start, end, _ := request.Window.Instants()
	type interval struct{ start, end time.Time }
	busy := []interval{}
	for _, id := range request.CalendarIDs {
		c, found := byID[id]
		if !found {
			c = CalendarBusy{CalendarID: id, Busy: []Window{}, ErrorCodes: []string{"unavailable"}}
		}
		c.Busy = append([]Window{}, c.Busy...)
		c.ErrorCodes = append([]string(nil), c.ErrorCodes...)
		out.Complete = out.Complete && c.Complete && len(c.ErrorCodes) == 0
		for _, w := range c.Busy {
			a, b, err := w.Instants()
			if err != nil {
				return Availability{}, err
			}
			if a.Before(start) {
				a = start
			}
			if b.After(end) {
				b = end
			}
			if b.After(a) {
				busy = append(busy, interval{a, b})
			}
		}
		out.Calendars = append(out.Calendars, c)
	}
	if len(byID) > len(request.CalendarIDs) {
		return Availability{}, fmt.Errorf("unexpected calendar availability")
	}
	for id := range byID {
		found := false
		for _, requested := range request.CalendarIDs {
			found = found || id == requested
		}
		if !found {
			return Availability{}, fmt.Errorf("unexpected calendar availability")
		}
	}
	if !out.Complete {
		return out, nil
	}
	sort.Slice(busy, func(i, j int) bool { return busy[i].start.Before(busy[j].start) })
	add := func(a, b time.Time) {
		out.Free = append(out.Free, Window{Start: a.UTC().Format(time.RFC3339Nano), End: b.UTC().Format(time.RFC3339Nano)})
	}
	cursor := start
	for _, b := range busy {
		if b.start.After(cursor) {
			add(cursor, b.start)
		}
		if b.end.After(cursor) {
			cursor = b.end
		}
	}
	if cursor.Before(end) {
		add(cursor, end)
	}
	return out, nil
}
