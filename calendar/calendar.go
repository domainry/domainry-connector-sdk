// Package calendar defines provider-neutral read contracts. It owns no account,
// credential, authorization, persistence or network implementation.
package calendar

type PageRequest struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// Window uses absolute instants and a half-open [start,end) interval.
type Window struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type EventsRequest struct {
	CalendarID string `json:"calendar_id"`
	Window     Window `json:"window"`
	TimeZone   string `json:"time_zone"`
	Limit      int    `json:"limit,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
}

type EventRequest struct {
	CalendarID string `json:"calendar_id"`
	EventID    string `json:"event_id"`
	TimeZone   string `json:"time_zone"`
}

type AvailabilityRequest struct {
	CalendarIDs []string `json:"calendar_ids"`
	Window      Window   `json:"window"`
	TimeZone    string   `json:"time_zone"`
}

type Calendar struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	TimeZone   string `json:"time_zone,omitempty"`
	Primary    bool   `json:"primary"`
	AccessRole string `json:"access_role,omitempty"`
}

// Moment is either a calendar date (all-day) or an RFC3339 instant. All-day end
// dates are exclusive. Dates must never be converted to midnight UTC instants.
// TimeZone preserves source zone metadata; DateTime always has its own offset.
type Moment struct {
	Date     string `json:"date,omitempty"`
	DateTime string `json:"date_time,omitempty"`
	TimeZone string `json:"time_zone,omitempty"`
}

type Event struct {
	ID            string  `json:"id"`
	CalendarID    string  `json:"calendar_id"`
	Title         string  `json:"title,omitempty"`
	Description   string  `json:"description,omitempty"`
	Location      string  `json:"location,omitempty"`
	URL           string  `json:"url,omitempty"`
	Status        string  `json:"status,omitempty"`
	Start         Moment  `json:"start"`
	End           Moment  `json:"end"`
	SeriesID      string  `json:"series_id,omitempty"`
	OriginalStart *Moment `json:"original_start,omitempty"`
	Transparency  string  `json:"transparency,omitempty"`
}

type CalendarsPage struct {
	Items      []Calendar `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
	Complete   bool       `json:"complete"`
}

type EventsPage struct {
	Items      []Event `json:"items"`
	NextCursor string  `json:"next_cursor,omitempty"`
	Complete   bool    `json:"complete"`
	TimeZone   string  `json:"time_zone,omitempty"`
}

// Complete requires a positive result for every requested calendar. Empty Busy
// alone is not evidence of availability. Unknown results cannot produce Free.
type CalendarBusy struct {
	CalendarID string   `json:"calendar_id"`
	Busy       []Window `json:"busy"`
	Complete   bool     `json:"complete"`
	ErrorCodes []string `json:"error_codes,omitempty"`
}

type Availability struct {
	Window    Window         `json:"window"`
	Calendars []CalendarBusy `json:"calendars"`
	Free      []Window       `json:"free"`
	Complete  bool           `json:"complete"`
}
