package calendar

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestContractIdentity(t *testing.T) {
	if got := ComputedContractSHA256(); got != ContractSHA256 {
		t.Fatalf("calendar contract hash=%s", got)
	}
	seen := map[string]bool{}
	for _, key := range []string{ListOperationKey, EventsOperationKey, EventOperationKey, AvailabilityOperationKey} {
		hash := OperationSHA256(key)
		if len(hash) != 64 || seen[hash] {
			t.Fatal("operation identities are not distinct")
		}
		seen[hash] = true
	}
	if OperationSHA256("calendar_send") != "" {
		t.Fatal("unknown operation has identity")
	}
}

func TestCalendarTimeAndQueryValidation(t *testing.T) {
	query := EventsRequest{CalendarID: "calendar@example.test", Window: Window{"2026-03-08T00:00:00-08:00", "2026-03-09T00:00:00-07:00"}, TimeZone: "America/Los_Angeles"}
	if err := query.Validate(); err != nil {
		t.Fatal(err)
	}
	a, b, _ := query.Window.Instants()
	if b.Sub(a) != 23*time.Hour {
		t.Fatal("DST window lost its offsets")
	}
	for _, change := range []func(*EventsRequest){
		func(q *EventsRequest) { q.Window.Start = "2026-03-08T00:00:00" },
		func(q *EventsRequest) { q.Window.End = q.Window.Start },
		func(q *EventsRequest) { q.Window.End = "2027-03-09T00:00:00Z" },
		func(q *EventsRequest) { q.TimeZone = "Local" },
		func(q *EventsRequest) { q.TimeZone = "unknown/zone" },
		func(q *EventsRequest) { q.Limit = 101 },
		func(q *EventsRequest) { q.CalendarID = "bad\ncalendar" },
		func(q *EventsRequest) { q.CalendarID = ".." },
	} {
		copy := query
		change(&copy)
		if copy.Validate() == nil {
			t.Fatalf("invalid query accepted: %+v", copy)
		}
	}
	e := Event{ID: "all-day", CalendarID: "calendar", Start: Moment{Date: "2026-03-08"}, End: Moment{Date: "2026-03-10"}}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(e)
	var got Event
	if err := json.Unmarshal(raw, &got); err != nil || !reflect.DeepEqual(e, got) {
		t.Fatal("date-only event changed")
	}
	e.End.Date = "2026-03-08"
	if e.Validate() == nil {
		t.Fatal("inclusive all-day end accepted")
	}
	e.End = Moment{DateTime: "2026-03-09T00:00:00Z"}
	if e.Validate() == nil {
		t.Fatal("mixed all-day/instant accepted")
	}
	for _, m := range []Moment{{}, {Date: "2026-02-30"}, {Date: "2026-03-08", DateTime: "2026-03-08T00:00:00Z"}, {DateTime: "2026-03-08T02:30:00"}} {
		if m.Validate() == nil {
			t.Fatalf("ambiguous moment accepted: %+v", m)
		}
	}
}

func TestLocalCalendarTimeRequiresUniqueInstant(t *testing.T) {
	for _, tc := range []struct{ value, zone, want string }{
		{"2026-09-11T10:00:00.1234567", "Asia/Shanghai", "2026-09-11T10:00:00.1234567+08:00"},
		{"2026-03-08T02:30:00", "America/Los_Angeles", ""},
		{"2026-11-01T01:30:00", "America/Los_Angeles", ""},
		{"2026-03-08T03:30:00", "America/Los_Angeles", "2026-03-08T03:30:00-07:00"},
		{"2026-04-05T01:45:00", "Australia/Lord_Howe", ""},
		{"2026-09-11T10:00:00", "UTC", "2026-09-11T10:00:00Z"},
	} {
		got, err := LocalDateTime(tc.value, tc.zone)
		if got != tc.want || (err != nil) != (tc.want == "") {
			t.Fatalf("local %s %s got=%s err=%v", tc.value, tc.zone, got, err)
		}
	}
}

func TestAvailabilityClipsMergesAndNeverInfersUnknownAsFree(t *testing.T) {
	r := AvailabilityRequest{CalendarIDs: []string{"a", "b"}, Window: Window{"2026-09-11T09:00:00+08:00", "2026-09-11T18:00:00+08:00"}, TimeZone: "Asia/Shanghai"}
	w := func(start, end string) Window {
		return Window{"2026-09-11T" + start + ":00+08:00", "2026-09-11T" + end + ":00+08:00"}
	}
	values := []CalendarBusy{{CalendarID: "a", Complete: true, Busy: []Window{w("14:00", "15:00"), w("08:00", "10:00")}}, {CalendarID: "b", Complete: true, Busy: []Window{w("09:30", "11:00"), w("15:00", "16:00"), w("17:00", "19:00")}}}
	got, err := ResolveAvailability(r, values)
	if err != nil || !got.Complete {
		t.Fatal(got, err)
	}
	want := []Window{{"2026-09-11T03:00:00Z", "2026-09-11T06:00:00Z"}, {"2026-09-11T08:00:00Z", "2026-09-11T09:00:00Z"}}
	if !reflect.DeepEqual(got.Free, want) {
		t.Fatalf("free=%+v", got.Free)
	}
	got.Calendars[0].Busy[0].Start = "mutated"
	if values[0].Busy[0].Start == "mutated" {
		t.Fatal("result aliases provider input")
	}
	for _, input := range [][]CalendarBusy{values[:1], {{CalendarID: "a", Complete: true}, {CalendarID: "b", Complete: false}}, {{CalendarID: "a", Complete: true}, {CalendarID: "b", Complete: true, ErrorCodes: []string{"forbidden"}}}} {
		got, err = ResolveAvailability(r, input)
		if err != nil || got.Complete || len(got.Free) != 0 {
			t.Fatal("partial availability became free", got, err)
		}
	}
	got, err = ResolveAvailability(r, []CalendarBusy{{CalendarID: "a", Complete: true, Busy: []Window{}}, {CalendarID: "b", Complete: true, Busy: []Window{}}})
	if err != nil || !got.Complete || len(got.Free) != 1 {
		t.Fatal("explicitly free window lost", got, err)
	}
	if _, err = ResolveAvailability(r, []CalendarBusy{{CalendarID: "foreign", Complete: true}}); err == nil {
		t.Fatal("foreign calendar accepted")
	}
	if _, err = ResolveAvailability(r, []CalendarBusy{{CalendarID: "a", Complete: true}, {CalendarID: "a", Complete: true}}); err == nil {
		t.Fatal("duplicate calendar accepted")
	}
}
