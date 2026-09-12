package calendarwrite_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/domainry/domainry-connector-sdk/calendar"
	write "github.com/domainry/domainry-connector-sdk/calendarwrite"
)

func draft() write.Draft {
	return write.Draft{Title: "项目评审", Start: calendar.Moment{DateTime: "2026-09-12T10:00:00+08:00", TimeZone: "Asia/Shanghai"}, End: calendar.Moment{DateTime: "2026-09-12T11:00:00+08:00", TimeZone: "Asia/Shanghai"}, Attendees: []write.Attendee{{Address: "person@example.test", Name: "同事", Kind: "required"}}}
}

func snapshot() write.Snapshot {
	d := draft()
	return write.Snapshot{Event: calendar.Event{ID: "event", CalendarID: "calendar", Title: d.Title, Start: d.Start, End: d.End, Status: "confirmed"}, Version: `"version-1"`, Kind: "single", Attendees: d.Attendees}
}

func TestContractIdentityAndReadIsolation(t *testing.T) {
	if got := write.ComputedContractSHA256(); got != write.ContractSHA256 {
		t.Fatalf("calendar write contract SHA256=%s", got)
	}
	seen := map[string]bool{}
	for _, key := range []string{write.InspectOperationKey, write.CreateOperationKey, write.UpdateOperationKey} {
		hash := write.OperationSHA256(key)
		if len(hash) != 64 || seen[hash] || calendar.OperationSHA256(key) != "" {
			t.Fatal("calendar write contract is not distinct", key, hash)
		}
		seen[hash] = true
	}
	if write.OperationSHA256(calendar.EventOperationKey) != "" || write.OperationSHA256("calendar_event_delete") != "" || calendar.ComputedContractSHA256() != calendar.ContractSHA256 {
		t.Fatal("read or undeclared operations acquired write identity")
	}
}

func TestCalendarCreateValidationPreservesExplicitTimeAndRecipients(t *testing.T) {
	request := func() write.CreateRequest {
		return write.CreateRequest{CalendarID: "calendar", Event: draft(), Notifications: write.NotifyAttendees}
	}
	if err := request().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*write.CreateRequest){
		"missing-calendar":     func(r *write.CreateRequest) { r.CalendarID = "" },
		"path-traversal":       func(r *write.CreateRequest) { r.CalendarID = "calendar/../other" },
		"invalid-utf8":         func(r *write.CreateRequest) { r.CalendarID = string([]byte{0xff}) },
		"silent-notifications": func(r *write.CreateRequest) { r.Notifications = "none" },
		"blank-title":          func(r *write.CreateRequest) { r.Event.Title = " " },
		"large-title":          func(r *write.CreateRequest) { r.Event.Title = strings.Repeat("中", 342) },
		"large-body": func(r *write.CreateRequest) {
			r.Event.Description = strings.Repeat("a", write.MaximumDescriptionBytes+1)
		},
		"body-control":      func(r *write.CreateRequest) { r.Event.Description = "x\x00y" },
		"missing-zone":      func(r *write.CreateRequest) { r.Event.Start.TimeZone = "" },
		"local-zone":        func(r *write.CreateRequest) { r.Event.Start.TimeZone = "Local" },
		"missing-offset":    func(r *write.CreateRequest) { r.Event.Start.DateTime = "2026-09-12T10:00:00" },
		"wrong-offset":      func(r *write.CreateRequest) { r.Event.Start.DateTime = "2026-09-12T10:00:00Z" },
		"zero-duration":     func(r *write.CreateRequest) { r.Event.End = r.Event.Start },
		"reversed-duration": func(r *write.CreateRequest) { r.Event.End.DateTime = "2026-09-12T09:00:00+08:00" },
		"mixed-bounds": func(r *write.CreateRequest) {
			r.Event.End = calendar.Moment{Date: "2026-09-13", TimeZone: "Asia/Shanghai"}
		},
		"unknown-attendees": func(r *write.CreateRequest) { r.Event.Attendees = nil },
		"duplicate-attendees": func(r *write.CreateRequest) {
			r.Event.Attendees = append(r.Event.Attendees, write.Attendee{Address: "Person@example.test", Kind: "optional"})
		},
		"attendee-injection": func(r *write.CreateRequest) {
			r.Event.Attendees[0].Address = "one@example.test\r\nBcc: two@example.test"
		},
		"multiple-addresses":    func(r *write.CreateRequest) { r.Event.Attendees[0].Address = "one@example.test,two@example.test" },
		"embedded-display-name": func(r *write.CreateRequest) { r.Event.Attendees[0].Address = "Other <one@example.test>" },
		"unknown-kind":          func(r *write.CreateRequest) { r.Event.Attendees[0].Kind = "bcc" },
	} {
		t.Run(name, func(t *testing.T) {
			r := request()
			change(&r)
			if r.Validate() == nil {
				t.Fatal("invalid mutation accepted")
			}
		})
	}
	for _, times := range [][2]calendar.Moment{
		{{Date: "2026-03-08", TimeZone: "America/Los_Angeles"}, {Date: "2026-03-09", TimeZone: "America/Los_Angeles"}},
		{{DateTime: "2026-03-08T01:30:00-08:00", TimeZone: "America/Los_Angeles"}, {DateTime: "2026-03-08T03:30:00-07:00", TimeZone: "America/Los_Angeles"}},
		{{DateTime: "2026-11-01T01:30:00-07:00", TimeZone: "America/Los_Angeles"}, {DateTime: "2026-11-01T01:30:00-08:00", TimeZone: "America/Los_Angeles"}},
	} {
		r := request()
		r.Event.Start, r.Event.End, r.Event.Attendees = times[0], times[1], []write.Attendee{}
		if err := r.Validate(); err != nil {
			t.Fatal("explicit DST bounds rejected", times, err)
		}
	}
	r := request()
	r.Event.Start = calendar.Moment{DateTime: "2026-03-08T02:30:00-08:00", TimeZone: "America/Los_Angeles"}
	if r.Validate() == nil {
		t.Fatal("nonexistent local time was silently normalized")
	}
	r.Event.Start = calendar.Moment{Date: "2026-09-12", TimeZone: "Asia/Shanghai"}
	r.Event.End = calendar.Moment{Date: "2026-09-13", TimeZone: "UTC"}
	if r.Validate() == nil {
		t.Fatal("all-day event crossed time zones")
	}
}

func TestUpdatePatchDistinguishesOmittedAndClearedFields(t *testing.T) {
	empty, none := "", []write.Attendee{}
	patch := write.Patch{Description: &empty, Attendees: &none}
	if err := patch.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(patch)
	if string(raw) != `{"description":"","attendees":[]}` {
		t.Fatal("patch overwrote omitted fields or lost explicit clear", string(raw))
	}
	var restored write.Patch
	if err := json.Unmarshal(raw, &restored); err != nil || restored.Validate() != nil || restored.Title != nil || restored.Description == nil || *restored.Description != "" || restored.Attendees == nil || len(*restored.Attendees) != 0 {
		t.Fatal("patch clear did not survive JSON")
	}
	var unknown []write.Attendee
	start := draft().Start
	for _, invalid := range []write.Patch{{}, {Attendees: &unknown}, {Start: &start}} {
		if invalid.Validate() == nil {
			t.Fatal("empty or incomplete patch accepted")
		}
	}
}

func TestMutationJSONRejectsUndeclaredAuthorityAndVendorFields(t *testing.T) {
	request := write.CreateRequest{CalendarID: "calendar", Event: draft(), Notifications: write.NotifyAttendees}
	raw, _ := json.Marshal(request)
	var got write.CreateRequest
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, addition := range []string{`"access_token":"secret"`, `"request_ref":"chosen"`, `"sql":"select 1"`, `"recurrence":["RRULE:FREQ=DAILY"]`} {
		modified := append(append([]byte{}, raw[:len(raw)-1]...), []byte(","+addition+"}")...)
		if json.Unmarshal(modified, &got) == nil {
			t.Fatal("undeclared request field accepted", addition)
		}
	}
	for _, raw := range []string{"null", "{}", strings.Repeat(" ", write.MaximumRequestBytes+1), `{"calendar_id":"calendar","event":{"title":"x","raw_vendor_data":{}},"notifications":"notify_attendees"}`} {
		if json.Unmarshal([]byte(raw), &got) == nil {
			t.Fatal("invalid request object accepted")
		}
	}
	title := "changed"
	update := write.UpdateRequest{CalendarID: "calendar", EventID: "event", ExpectedVersion: `"version"`, Scope: write.ScopeEvent, Changes: write.Patch{Title: &title}, Notifications: write.NotifyAttendees}
	raw, _ = json.Marshal(update)
	var updated write.UpdateRequest
	if err := json.Unmarshal(raw, &updated); err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal([]byte(strings.Replace(string(raw), `"title":"changed"`, `"title":"changed","native_body":{}`, 1)), &updated) == nil {
		t.Fatal("nested undeclared patch accepted")
	}
}

func TestMaximumCalendarTextAndAttendeesSurviveJSONEscaping(t *testing.T) {
	d := draft()
	d.Title = strings.Repeat("<", write.MaximumTitleBytes)
	d.Description = strings.Repeat("<", write.MaximumDescriptionBytes)
	d.Location = strings.Repeat("<", write.MaximumLocationBytes)
	d.Attendees = []write.Attendee{}
	for i := 0; i < write.MaximumAttendees; i++ {
		d.Attendees = append(d.Attendees, write.Attendee{Address: fmt.Sprintf("person%d@example.test", i), Name: strings.Repeat("<", 512), Kind: "required"})
	}
	r := write.CreateRequest{CalendarID: "calendar", Event: d, Notifications: write.NotifyAttendees}
	raw, err := json.Marshal(r)
	var decoded write.CreateRequest
	if err != nil || len(raw) > write.MaximumRequestBytes || json.Unmarshal(raw, &decoded) != nil || decoded.Event.Description != d.Description || len(decoded.Event.Attendees) != write.MaximumAttendees {
		t.Fatal("valid bounded event failed its escaped JSON round trip", len(raw), err)
	}
	r.Event.Attendees = append(r.Event.Attendees, write.Attendee{Address: "extra@example.test", Kind: "optional"})
	if r.Validate() == nil {
		t.Fatal("calendar attendee limit was not enforced")
	}
}

func TestUpdateBindsCurrentTargetVersionAndSeriesScope(t *testing.T) {
	title := "调整评审主题"
	s := snapshot()
	r := write.UpdateRequest{CalendarID: "calendar", EventID: "event", ExpectedVersion: s.Version, Scope: write.ScopeEvent, Changes: write.Patch{Title: &title}, Notifications: write.NotifyAttendees}
	if err := r.ValidateAgainst(s); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*write.Snapshot){
		"different-version":  func(s *write.Snapshot) { s.Version = `"version-2"` },
		"different-calendar": func(s *write.Snapshot) { s.Event.CalendarID = "other" },
		"different-event":    func(s *write.Snapshot) { s.Event.ID = "other" },
		"unexpected-series":  func(s *write.Snapshot) { s.Kind = "series" },
		"cancelled":          func(s *write.Snapshot) { s.Event.Status = "cancelled" },
		"unknown-status":     func(s *write.Snapshot) { s.Event.Status = "" },
		"unknown-kind":       func(s *write.Snapshot) { s.Kind = "unknown" },
		"partial-attendees":  func(s *write.Snapshot) { s.Attendees = nil },
		"unsafe-link":        func(s *write.Snapshot) { s.Event.URL = "javascript:alert(1)" },
	} {
		t.Run(name, func(t *testing.T) {
			current := snapshot()
			change(&current)
			if r.ValidateAgainst(current) == nil {
				t.Fatal("changed or incomplete target accepted")
			}
		})
	}
	s.Kind, r.Scope = "series", write.ScopeSeries
	if err := r.ValidateAgainst(s); err != nil {
		t.Fatal("explicit series change rejected", err)
	}
	s.Kind, s.Event.SeriesID, s.Event.OriginalStart, r.Scope = "occurrence", "master", &s.Event.Start, write.ScopeEvent
	if err := r.ValidateAgainst(s); err != nil {
		t.Fatal("explicit occurrence change rejected", err)
	}
	for _, version := range []string{"", "*", `W/*`, `"a", "b"`, "\"a\r\nb\"", `""`, `version-1`} {
		if write.ValidVersion(version) {
			t.Fatal("invalid entity tag accepted", version)
		}
	}
	for _, version := range []string{`"version-1"`, `W/"base64=="`} {
		if !write.ValidVersion(version) {
			t.Fatal("actual entity tag rejected", version)
		}
	}
}

func TestReceiptCannotClaimAnotherMutationOrNotificationDelivery(t *testing.T) {
	r := write.Result{RequestRef: "host-request", Outcome: "updated", CalendarID: "calendar", EventID: "event", Version: `W/"new-version"`, Notifications: "requested", URL: "https://calendar.example.test/event"}
	if err := r.Validate(write.UpdateOperationKey, "host-request", "calendar", "event"); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*write.Result){
		"different-request":   func(r *write.Result) { r.RequestRef = "other" },
		"different-calendar":  func(r *write.Result) { r.CalendarID = "other" },
		"different-event":     func(r *write.Result) { r.EventID = "other" },
		"no-version":          func(r *write.Result) { r.Version = "" },
		"created-not-updated": func(r *write.Result) { r.Outcome = "created" },
		"delivery-not-proven": func(r *write.Result) { r.Notifications = "delivered" },
		"unsafe-url":          func(r *write.Result) { r.URL = "https://user:password@example.test/event" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := r
			change(&changed)
			if changed.Validate(write.UpdateOperationKey, "host-request", "calendar", "event") == nil {
				t.Fatal("invalid receipt accepted")
			}
		})
	}
	r.Outcome = "created"
	if r.Validate(write.CreateOperationKey, "host-request", "calendar", "") != nil || r.Validate(write.CreateOperationKey, "host-request", "calendar", "other") == nil {
		t.Fatal("create receipt does not bind expected target")
	}
}
