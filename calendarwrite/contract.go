// Package calendarwrite defines bounded calendar mutation contracts. Calendar
// records, account authority, confirmation and durable receipts belong to their
// owners. No request carries credentials or a caller-selected idempotency key.
package calendarwrite

import "github.com/domainry/domainry-connector-sdk/calendar"

const (
	InspectOperationKey = "calendar_event_inspect"
	CreateOperationKey  = "calendar_event_create"
	UpdateOperationKey  = "calendar_event_update"
	NotifyAttendees     = "notify_attendees"
	ScopeEvent          = "event"
	ScopeSeries         = "series"
)

// Attendees are the complete invitation list, not an appended set. Providers
// must reject unsupported kinds rather than silently changing the recipient.
type Attendee struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
	Kind    string `json:"kind"` // required, optional, resource
}

type InspectRequest struct {
	CalendarID string `json:"calendar_id"`
	EventID    string `json:"event_id"`
	TimeZone   string `json:"time_zone"`
}

// Snapshot supplies the actual version and the target event's complete
// attendee list. Providers cannot truncate it. A series edit can additionally
// notify separately changed occurrences; previews must explain that scope.
// Kind is single, occurrence, exception or series; a series edit is explicit.
type Snapshot struct {
	Event     calendar.Event `json:"event"`
	Version   string         `json:"version"`
	Kind      string         `json:"kind"`
	Attendees []Attendee     `json:"attendees"`
}

// Draft creates a single event. Dates retain their exclusive end and named
// zone; instants include both an offset and the matching named time zone.
// Description is plain text. Recurrence and attachments are separate vendor
// features and cannot be smuggled into this typed input.
type Draft struct {
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Location    string          `json:"location,omitempty"`
	Start       calendar.Moment `json:"start"`
	End         calendar.Moment `json:"end"`
	Attendees   []Attendee      `json:"attendees"`
}

type CreateRequest struct {
	CalendarID    string `json:"calendar_id"`
	Event         Draft  `json:"event"`
	Notifications string `json:"notifications"`
}

// Nil omits a field; an empty description/location or attendee array clears
// that field. Start and End must be supplied together. Other fields remain
// owned by the provider; updates must use PATCH, never full replacement.
type Patch struct {
	Title       *string          `json:"title,omitempty"`
	Description *string          `json:"description,omitempty"`
	Location    *string          `json:"location,omitempty"`
	Start       *calendar.Moment `json:"start,omitempty"`
	End         *calendar.Moment `json:"end,omitempty"`
	Attendees   *[]Attendee      `json:"attendees,omitempty"`
}

type UpdateRequest struct {
	CalendarID      string `json:"calendar_id"`
	EventID         string `json:"event_id"`
	ExpectedVersion string `json:"expected_version"`
	Scope           string `json:"scope"`
	Changes         Patch  `json:"changes"`
	Notifications   string `json:"notifications"`
}

// Result acknowledges the exact mutation, never delivery of invitations.
// The provider obtains RequestRef from the trusted invocation envelope. The
// Integration owner persists this bounded receipt without storing event text.
type Result struct {
	RequestRef    string `json:"request_ref"`
	Outcome       string `json:"outcome"` // created or updated
	CalendarID    string `json:"calendar_id"`
	EventID       string `json:"event_id"`
	Version       string `json:"version"`
	URL           string `json:"url,omitempty"`
	Notifications string `json:"notifications"` // requested, never delivered
}
