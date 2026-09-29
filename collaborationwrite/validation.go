package collaborationwrite

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func validText(value string, maximum int, allowNewline bool) bool {
	if value == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) || utf8.RuneCountInString(value) > maximum {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) && (!allowNewline || char != '\n' && char != '\r' && char != '\t') {
			return false
		}
	}
	return true
}

func (request SendRequest) Validate() error {
	if !validText(request.Recipient, 320, false) || !validText(request.Text, 8000, true) {
		return fmt.Errorf("collaboration message recipient and text are invalid")
	}
	// The connection declares whether this one exact identifier is an email,
	// open_id, union_id, user_id or chat_id. Keep the shared write contract
	// provider-neutral while rejecting ambiguous whitespace-delimited values.
	if strings.IndexFunc(request.Recipient, unicode.IsSpace) >= 0 {
		return fmt.Errorf("collaboration message recipient must be one identifier")
	}
	return nil
}

func (result Result) Validate(requestRef string) error {
	if result.RequestRef != requestRef || !validText(result.RequestRef, 2048, false) || result.Status != "accepted" || !validText(result.MessageID, 512, false) || result.Delivery != "unknown" {
		return fmt.Errorf("collaboration message receipt is invalid")
	}
	accepted, err := time.Parse(time.RFC3339Nano, result.AcceptedAt)
	if err != nil || accepted.IsZero() {
		return fmt.Errorf("collaboration message receipt time is invalid")
	}
	return nil
}
