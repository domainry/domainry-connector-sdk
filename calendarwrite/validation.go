package calendarwrite

import (
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/domainry/domainry-connector-sdk/calendar"
)

const (
	MaximumAttendees        = 100
	MaximumTitleBytes       = 1024
	MaximumDescriptionBytes = 64 << 10
	MaximumLocationBytes    = 4096
)

func text(value string, limit int, multiline bool) bool {
	return len(value) <= limit && utf8.ValidString(value) && !strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\r' || r == '\t'))
	})
}

func validID(value string) bool {
	if !calendar.ValidID(value) || !text(value, 1024, false) {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}

func validLink(value string) bool {
	if value == "" {
		return true
	}
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && text(value, 8192, false)
}

// ValidVersion accepts one opaque HTTP entity tag, including Graph weak tags.
// Lists and wildcards would weaken a specific record's optimistic guard.
func ValidVersion(value string) bool {
	if len(value) > 1024 {
		return false
	}
	value = strings.TrimPrefix(value, "W/")
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' {
		return false
	}
	for _, c := range value[1 : len(value)-1] {
		if c < 0x21 || c > 0x7e || c == '"' {
			return false
		}
	}
	return true
}

func (a Attendee) Validate() error {
	parsed, err := mail.ParseAddress(a.Address)
	if err != nil || parsed.Address != a.Address || parsed.Name != "" || !text(a.Address, 320, false) || !text(a.Name, 512, false) {
		return errors.New("calendar attendee must contain one explicit mailbox address")
	}
	if a.Kind != "required" && a.Kind != "optional" && a.Kind != "resource" {
		return errors.New("calendar attendee kind is invalid")
	}
	return nil
}

func attendees(values []Attendee) error {
	if values == nil || len(values) > MaximumAttendees {
		return errors.New("calendar attendees must be a complete array of at most 100 entries")
	}
	seen := map[string]bool{}
	for _, a := range values {
		if err := a.Validate(); err != nil {
			return err
		}
		key := strings.ToLower(a.Address)
		if seen[key] {
			return errors.New("calendar attendee is duplicated")
		}
		seen[key] = true
	}
	return nil
}

func moment(value calendar.Moment) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.TimeZone == "" || value.TimeZone == "Local" || len(value.TimeZone) > 128 {
		return errors.New("calendar write requires an explicit IANA time zone")
	}
	zone, err := time.LoadLocation(value.TimeZone)
	if err != nil {
		return errors.New("calendar write time zone is invalid")
	}
	if value.DateTime != "" {
		instant, _ := time.Parse(time.RFC3339, value.DateTime)
		_, supplied := instant.Zone()
		_, actual := instant.In(zone).Zone()
		if supplied != actual {
			return errors.New("calendar write offset does not match its time zone")
		}
	}
	return nil
}

func bounds(start, end calendar.Moment) error {
	if err := moment(start); err != nil {
		return err
	}
	if err := moment(end); err != nil {
		return err
	}
	if (start.Date == "") != (end.Date == "") {
		return errors.New("calendar write mixes dates and instants")
	}
	if start.Date != "" {
		if start.TimeZone != end.TimeZone || start.Date >= end.Date {
			return errors.New("all-day write requires one zone and an exclusive later end date")
		}
	} else {
		a, _ := time.Parse(time.RFC3339, start.DateTime)
		b, _ := time.Parse(time.RFC3339, end.DateTime)
		if !b.After(a) {
			return errors.New("calendar write end must be after start")
		}
	}
	return nil
}

func (r InspectRequest) Validate() error {
	if !validID(r.CalendarID) || !validID(r.EventID) {
		return errors.New("calendar inspection target is invalid")
	}
	return (calendar.EventRequest{CalendarID: r.CalendarID, EventID: r.EventID, TimeZone: r.TimeZone}).Validate()
}

func (s Snapshot) Validate(r InspectRequest) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := s.Event.Validate(); err != nil {
		return err
	}
	if s.Event.CalendarID != r.CalendarID || s.Event.ID != r.EventID || !ValidVersion(s.Version) || s.Event.Status != "confirmed" && s.Event.Status != "tentative" || !validLink(s.Event.URL) {
		return errors.New("calendar snapshot does not identify the current event")
	}
	if !text(s.Event.Title, MaximumTitleBytes, false) || !text(s.Event.Description, MaximumDescriptionBytes, true) || !text(s.Event.Location, MaximumLocationBytes, false) {
		return errors.New("calendar snapshot text exceeds the preview contract")
	}
	switch s.Kind {
	case "single", "series":
		if s.Event.SeriesID != "" || s.Event.OriginalStart != nil {
			return errors.New("calendar snapshot has inconsistent series identity")
		}
	case "occurrence", "exception":
		if !validID(s.Event.SeriesID) || s.Event.SeriesID == s.Event.ID || s.Event.OriginalStart == nil {
			return errors.New("calendar occurrence requires its series and original start")
		}
	default:
		return errors.New("calendar snapshot kind is unknown")
	}
	return attendees(s.Attendees)
}

func (d Draft) Validate() error {
	if strings.TrimSpace(d.Title) == "" || !text(d.Title, MaximumTitleBytes, false) || !text(d.Description, MaximumDescriptionBytes, true) || !text(d.Location, MaximumLocationBytes, false) {
		return errors.New("calendar draft text is invalid")
	}
	if err := bounds(d.Start, d.End); err != nil {
		return err
	}
	return attendees(d.Attendees)
}

func (r CreateRequest) Validate() error {
	if !validID(r.CalendarID) || r.Notifications != NotifyAttendees {
		return errors.New("calendar create requires an exact calendar and notification policy")
	}
	return r.Event.Validate()
}

func (p Patch) Validate() error {
	if p.Title == nil && p.Description == nil && p.Location == nil && p.Start == nil && p.End == nil && p.Attendees == nil {
		return errors.New("calendar update requires a change")
	}
	if p.Title != nil && (strings.TrimSpace(*p.Title) == "" || !text(*p.Title, MaximumTitleBytes, false)) || p.Description != nil && !text(*p.Description, MaximumDescriptionBytes, true) || p.Location != nil && !text(*p.Location, MaximumLocationBytes, false) {
		return errors.New("calendar update text is invalid")
	}
	if (p.Start == nil) != (p.End == nil) {
		return errors.New("calendar update must specify both time bounds")
	}
	if p.Start != nil {
		if err := bounds(*p.Start, *p.End); err != nil {
			return err
		}
	}
	if p.Attendees != nil {
		return attendees(*p.Attendees)
	}
	return nil
}

func (r UpdateRequest) Validate() error {
	if !validID(r.CalendarID) || !validID(r.EventID) || !ValidVersion(r.ExpectedVersion) || r.Notifications != NotifyAttendees || r.Scope != ScopeEvent && r.Scope != ScopeSeries {
		return errors.New("calendar update requires exact target, version, scope and notifications")
	}
	return r.Changes.Validate()
}

// ValidateAgainst checks a freshly retrieved preview, not a model assertion of
// current authority. The actual mutation still requires vendor If-Match.
func (r UpdateRequest) ValidateAgainst(s Snapshot) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := s.Validate(InspectRequest{CalendarID: r.CalendarID, EventID: r.EventID, TimeZone: "UTC"}); err != nil {
		return err
	}
	if r.ExpectedVersion != s.Version || (s.Kind == "series") != (r.Scope == ScopeSeries) {
		return errors.New("calendar event version or update scope changed")
	}
	return nil
}

func (r Result) Validate(operation, requestRef, calendarID, eventID string) error {
	outcome := map[string]string{CreateOperationKey: "created", UpdateOperationKey: "updated"}[operation]
	if outcome == "" || r.Outcome != outcome || requestRef == "" || r.RequestRef != requestRef || strings.TrimSpace(r.RequestRef) != r.RequestRef || !text(r.RequestRef, 2048, false) || r.CalendarID != calendarID || !validID(r.CalendarID) || !validID(r.EventID) || !ValidVersion(r.Version) || r.Notifications != "requested" {
		return errors.New("calendar mutation receipt is invalid")
	}
	if operation == UpdateOperationKey && eventID == "" || eventID != "" && r.EventID != eventID {
		return errors.New("calendar update receipt belongs to another event")
	}
	if !validLink(r.URL) {
		return errors.New("calendar receipt link is invalid")
	}
	return nil
}
