package mailwrite

import (
	"errors"
	netmail "net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/domainry/domainry-connector-sdk/mail"
)

const (
	MaximumRecipients   = 50
	MaximumSubjectBytes = 2048
	MaximumTextBytes    = 64 << 10
)

func text(value string, limit int, multiline bool) bool {
	return len(value) <= limit && utf8.ValidString(value) && !strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\r' || r == '\t'))
	})
}

func validAddress(value mail.Address) bool {
	parsed, err := netmail.ParseAddress(value.Address)
	return err == nil && parsed.Address == value.Address && parsed.Name == "" && text(value.Address, 320, false) && text(value.Name, 512, false)
}

func (m Message) Validate() error {
	if !text(m.Subject, MaximumSubjectBytes, false) || !text(m.Text, MaximumTextBytes, true) || strings.TrimSpace(m.Subject) == "" && strings.TrimSpace(m.Text) == "" {
		return errors.New("outgoing mail subject or text is invalid")
	}
	if m.To == nil || m.CC == nil || m.BCC == nil || len(m.To)+len(m.CC)+len(m.BCC) < 1 || len(m.To)+len(m.CC)+len(m.BCC) > MaximumRecipients {
		return errors.New("outgoing mail requires complete recipient arrays with 1 to 50 recipients")
	}
	seen := map[string]bool{}
	for _, group := range [][]mail.Address{m.To, m.CC, m.BCC} {
		for _, address := range group {
			key := strings.ToLower(address.Address)
			if !validAddress(address) || seen[key] {
				return errors.New("outgoing mail contains an invalid or duplicate recipient")
			}
			seen[key] = true
		}
	}
	return nil
}

func (r SendRequest) Validate() error { return r.Message.Validate() }

func (r ReplyRequest) Validate() error {
	if !mail.ValidID(r.MessageID) {
		return errors.New("mail reply requires an exact original message")
	}
	return r.Message.Validate()
}

func replySubject(value string) string {
	value = strings.TrimSpace(value)
	for len(value) >= 3 && strings.EqualFold(value[:3], "re:") {
		value = strings.TrimSpace(value[3:])
	}
	return value
}

// ValidateAgainst consumes actual, freshly read metadata, never a caller's
// assertion that the source remains readable. Providers additionally verify
// their native threading requirements before performing the send.
func (r ReplyRequest) ValidateAgainst(original mail.Summary) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := original.Validate(); err != nil {
		return err
	}
	if original.ID != r.MessageID || !original.MetadataComplete || original.IsDraft == nil || *original.IsDraft || replySubject(r.Message.Subject) != replySubject(original.Subject) {
		return errors.New("mail reply source or subject is incomplete or changed")
	}
	targets := original.ReplyTo
	if len(targets) == 0 {
		targets = original.From
	}
	if len(targets) == 0 {
		return errors.New("mail reply has no known original recipient")
	}
	for _, target := range targets {
		if !validAddress(target) {
			return errors.New("mail reply source recipient is invalid")
		}
		found := false
		for _, destination := range r.Message.To {
			found = found || strings.EqualFold(target.Address, destination.Address)
		}
		if !found {
			return errors.New("mail reply must explicitly include its original Reply-To or From targets")
		}
	}
	return nil
}

func (r Result) Validate(operation, requestRef, originalMessageID string) error {
	if operation != SendOperationKey && operation != ReplyOperationKey || r.Status != "accepted" || r.Delivery != "unknown" || requestRef == "" || r.RequestRef != requestRef || strings.TrimSpace(r.RequestRef) != r.RequestRef || !text(r.RequestRef, 2048, false) {
		return errors.New("mail acceptance receipt is invalid")
	}
	if r.MessageID != "" && !mail.ValidID(r.MessageID) || r.ThreadID != "" && (r.MessageID == "" || !mail.ValidID(r.ThreadID)) {
		return errors.New("mail acceptance identifiers are invalid")
	}
	if _, err := time.Parse(time.RFC3339Nano, r.AcceptedAt); err != nil {
		return errors.New("mail acceptance time requires an explicit offset")
	}
	if operation == SendOperationKey {
		if originalMessageID != "" || r.InReplyToMessageID != "" {
			return errors.New("new mail receipt cannot claim a reply")
		}
	} else if !mail.ValidID(originalMessageID) || r.InReplyToMessageID != originalMessageID {
		return errors.New("mail reply receipt belongs to another original message")
	}
	return nil
}
