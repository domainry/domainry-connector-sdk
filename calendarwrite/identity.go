package calendarwrite

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"

	"github.com/domainry/domainry-connector-sdk/calendar"
)

const ContractVersion = "calendar-write-v1"
const ContractSHA256 = "7535ffee48efcb9bdd8686ac25a5b3f8c9aa9be39cbe6a4298f603e0f66d5001"

func ComputedContractSHA256() string {
	var b strings.Builder
	b.WriteString(ContractVersion + ";calendar-read:" + calendar.ContractSHA256 + ";inspect-complete;create-single;patch-only;explicit-event-or-series;one-entity-tag;if-match-required;notify-attendees-explicit;notification-delivery-unknown;iana-offset-match;all-day-exclusive;positive-duration;attendees-100;title-1024;description-65536;location-4096;strict-json-1048576;request-ref-envelope-owned;uncertain-never-replay;")
	for _, value := range []any{Attendee{}, InspectRequest{}, Snapshot{}, Draft{}, CreateRequest{}, Patch{}, UpdateRequest{}, Result{}} {
		t := reflect.TypeOf(value)
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
	case InspectOperationKey, CreateOperationKey, UpdateOperationKey:
	default:
		return ""
	}
	sum := sha256.Sum256([]byte(ContractSHA256 + ":" + key))
	return hex.EncodeToString(sum[:])
}
