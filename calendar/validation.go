package calendar

import (
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"
	_ "time/tzdata"
)

const MaximumPageSize = 100
const MaximumCalendars = 20
const MaximumWindow = 93 * 24 * time.Hour
const MaximumParticipants = 1000
const MaximumRecurrenceRules = 64

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

func (p Participant) Validate() error {
	if p.ID == "" && p.Email == "" {
		return fmt.Errorf("calendar participant requires an id or email")
	}
	if p.ID != "" && !ValidID(p.ID) {
		return fmt.Errorf("invalid calendar participant id")
	}
	if p.Email != "" {
		address, err := mail.ParseAddress(p.Email)
		if err != nil || address.Address != p.Email || len(p.Email) > 320 {
			return fmt.Errorf("invalid calendar participant email")
		}
	}
	if len(p.DisplayName) > 1024 {
		return fmt.Errorf("calendar participant display name is too long")
	}
	switch p.Role {
	case "", "organizer", "required", "optional", "resource":
	default:
		return fmt.Errorf("invalid calendar participant role")
	}
	switch p.ResponseStatus {
	case "", "needs_action", "accepted", "tentative", "declined", "delegated", "unknown":
	default:
		return fmt.Errorf("invalid calendar participant response status")
	}
	return nil
}

func validateHTTPSURL(value, field string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || len(value) > 8192 {
		return fmt.Errorf("calendar %s must be an HTTPS URL", field)
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
		if err := e.OriginalStart.Validate(); err != nil {
			return err
		}
	}
	if err := validateHTTPSURL(e.MeetingURL, "meeting_url"); err != nil {
		return err
	}
	if len(e.Recurrence) > MaximumRecurrenceRules {
		return fmt.Errorf("calendar recurrence has too many rules")
	}
	for _, rule := range e.Recurrence {
		if strings.TrimSpace(rule) != rule || rule == "" || len(rule) > 8192 {
			return fmt.Errorf("invalid calendar recurrence rule")
		}
	}
	if e.Organizer != nil {
		if err := e.Organizer.Validate(); err != nil || e.Organizer.Role != "organizer" {
			return fmt.Errorf("invalid calendar organizer")
		}
	}
	if len(e.Attendees) > MaximumParticipants {
		return fmt.Errorf("calendar event has too many attendees")
	}
	for _, participant := range e.Attendees {
		if err := participant.Validate(); err != nil {
			return err
		}
	}
	return nil
}
