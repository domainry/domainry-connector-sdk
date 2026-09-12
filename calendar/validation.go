package calendar

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

const MaximumPageSize = 100
const MaximumCalendars = 20
const MaximumWindow = 93 * 24 * time.Hour

func ValidID(value string) bool {
	if value == "" || value == "." || value == ".." || len(value) > 1024 || strings.TrimSpace(value) != value {
		return false
	}
	for _, c := range value {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

func (p PageRequest) Validate() error {
	if p.Limit < 0 || p.Limit > MaximumPageSize || len(p.Cursor) > 8192 {
		return fmt.Errorf("invalid calendar page")
	}
	return nil
}

func (p PageRequest) PageSize() int {
	if p.Limit == 0 {
		return 25
	}
	return p.Limit
}

func (w Window) Instants() (time.Time, time.Time, error) {
	a, e1 := time.Parse(time.RFC3339, w.Start)
	b, e2 := time.Parse(time.RFC3339, w.End)
	if e1 != nil || e2 != nil || !b.After(a) {
		return time.Time{}, time.Time{}, fmt.Errorf("calendar window requires ordered RFC3339 instants")
	}
	return a, b, nil
}

func validateWindow(w Window, zone string) error {
	a, b, err := w.Instants()
	if err != nil {
		return err
	}
	if b.Sub(a) > MaximumWindow {
		return fmt.Errorf("calendar window exceeds 93 days")
	}
	return validateZone(zone)
}

func validateZone(zone string) error {
	if zone == "" || zone == "Local" || len(zone) > 128 {
		return fmt.Errorf("calendar time_zone must identify an IANA location")
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return fmt.Errorf("calendar time_zone must identify an IANA location")
	}
	return nil
}

func (r EventsRequest) Validate() error {
	if !ValidID(r.CalendarID) {
		return fmt.Errorf("calendar_id is required")
	}
	if err := (PageRequest{Limit: r.Limit, Cursor: r.Cursor}).Validate(); err != nil {
		return err
	}
	return validateWindow(r.Window, r.TimeZone)
}

func (r EventRequest) Validate() error {
	if !ValidID(r.CalendarID) || !ValidID(r.EventID) {
		return fmt.Errorf("calendar_id and event_id are required")
	}
	return validateZone(r.TimeZone)
}

func (r AvailabilityRequest) Validate() error {
	if len(r.CalendarIDs) < 1 || len(r.CalendarIDs) > MaximumCalendars {
		return fmt.Errorf("calendar_ids must contain 1 to 20 calendars")
	}
	seen := map[string]bool{}
	for _, id := range r.CalendarIDs {
		if !ValidID(id) || seen[id] {
			return fmt.Errorf("calendar_ids must contain unique identifiers")
		}
		seen[id] = true
	}
	return validateWindow(r.Window, r.TimeZone)
}

func (m Moment) Validate() error {
	if (m.Date == "") == (m.DateTime == "") {
		return fmt.Errorf("calendar moment requires exactly one date or date_time")
	}
	if m.Date != "" {
		if d, err := time.Parse(time.DateOnly, m.Date); err != nil || d.Format(time.DateOnly) != m.Date {
			return fmt.Errorf("invalid calendar date")
		}
	} else if _, err := time.Parse(time.RFC3339, m.DateTime); err != nil {
		return fmt.Errorf("calendar date_time requires an offset")
	}
	return nil
}

func (e Event) Validate() error {
	if !ValidID(e.ID) || !ValidID(e.CalendarID) {
		return fmt.Errorf("calendar event identifiers are required")
	}
	if err := e.Start.Validate(); err != nil {
		return err
	}
	if err := e.End.Validate(); err != nil {
		return err
	}
	if (e.Start.Date != "") != (e.End.Date != "") {
		return fmt.Errorf("calendar event mixes all-day and instant bounds")
	}
	if e.Start.Date != "" {
		if e.End.Date <= e.Start.Date {
			return fmt.Errorf("all-day event end must be exclusive")
		}
	} else {
		start, _ := time.Parse(time.RFC3339, e.Start.DateTime)
		end, _ := time.Parse(time.RFC3339, e.End.DateTime)
		if end.Before(start) {
			return fmt.Errorf("calendar event ends before it starts")
		}
	}
	if e.OriginalStart != nil {
		return e.OriginalStart.Validate()
	}
	return nil
}
