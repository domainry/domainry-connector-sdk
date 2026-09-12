package calendar

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
)

const ContractVersion = "calendar-read-v1"
const ContractSHA256 = "ef24ce63d41fce802f52702d991ca20c9f41148bcb7fdf310902fc22135288f6"
const (
	ListOperationKey         = "calendar_list"
	EventsOperationKey       = "calendar_events"
	EventOperationKey        = "calendar_event"
	AvailabilityOperationKey = "calendar_availability"
)

func ComputedContractSHA256() string {
	var b strings.Builder
	b.WriteString(ContractVersion + ";read-only;date-end-exclusive;instant-offset-required;window-half-open;max-window-93-days;page-100;calendars-20;unknown-never-free;local-time-iana-unique-or-reject;")
	for _, v := range []any{PageRequest{}, Window{}, EventsRequest{}, EventRequest{}, AvailabilityRequest{}, Calendar{}, Moment{}, Event{}, CalendarsPage{}, EventsPage{}, CalendarBusy{}, Availability{}} {
		t := reflect.TypeOf(v)
		b.WriteString(t.Name() + "{")
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			b.WriteString(f.Name + ":" + f.Type.String() + ":" + f.Tag.Get("json") + ";")
		}
		b.WriteString("}")
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func OperationSHA256(key string) string {
	switch key {
	case ListOperationKey, EventsOperationKey, EventOperationKey, AvailabilityOperationKey:
	default:
		return ""
	}
	sum := sha256.Sum256([]byte(ContractSHA256 + ":" + key))
	return hex.EncodeToString(sum[:])
}
