package mail

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultPageSize  = 10
	MaximumPageSize  = 25
	DefaultBodyBytes = 16 << 10
	MaximumBodyBytes = 64 << 10
	GmailSyntax      = "gmail"
	GraphSyntax      = "graph-kql"
)

func validText(s string, max int) bool {
	return len(s) <= max && utf8.ValidString(s) && !strings.ContainsFunc(s, unicode.IsControl)
}

func ValidID(s string) bool {
	if s == "" || strings.TrimSpace(s) != s || !validText(s, 2048) {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}
func (p PageRequest) PageSize() int {
	if p.Limit == 0 {
		return DefaultPageSize
	}
	return p.Limit
}
func (p PageRequest) Validate() error {
	if p.Limit < 0 || p.Limit > MaximumPageSize || !validText(p.Cursor, 8192) {
		return errors.New("invalid mail page")
	}
	return nil
}
func (s SearchRequest) Validate() error {
	if err := (PageRequest{Limit: s.Limit, Cursor: s.Cursor}).Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(s.Query) == "" || !validText(s.Query, 1024) || s.QuerySyntax != GmailSyntax && s.QuerySyntax != GraphSyntax {
		return errors.New("invalid mail search")
	}
	return nil
}
func (r ReadRequest) BodyLimit() int {
	if r.MaxBodyBytes == 0 {
		return DefaultBodyBytes
	}
	return r.MaxBodyBytes
}
func (r ReadRequest) Validate() error {
	if !ValidID(r.MessageID) || r.MaxBodyBytes != 0 && (r.MaxBodyBytes < 1024 || r.MaxBodyBytes > MaximumBodyBytes) {
		return errors.New("invalid mail read")
	}
	return nil
}

func (s Summary) Validate() error {
	if !ValidID(s.ID) || s.ThreadID != "" && !ValidID(s.ThreadID) || !validText(s.InternetMessageID, 2048) || !validText(s.Subject, 2048) {
		return errors.New("invalid mail metadata")
	}
	for _, date := range []string{s.ReceivedAt, s.SentAt} {
		if date != "" {
			if _, err := time.Parse(time.RFC3339Nano, date); err != nil {
				return errors.New("invalid mail date")
			}
		}
	}
	for _, list := range [][]Address{s.From, s.To, s.CC, s.ReplyTo} {
		if list == nil || len(list) > 50 {
			return errors.New("invalid mail addresses")
		}
		for _, a := range list {
			if a.Address == "" || !validText(a.Address, 320) || !validText(a.Name, 512) {
				return errors.New("invalid mail address")
			}
		}
	}
	return nil
}

func (p MessagesPage) Validate(limit int) error {
	if limit < 1 || limit > MaximumPageSize || p.Items == nil || len(p.Items) > limit || !validText(p.NextCursor, 8192) || p.QuerySyntax != GmailSyntax && p.QuerySyntax != GraphSyntax || !validText(p.MailboxScope, 128) || p.MailboxScope == "" {
		return errors.New("invalid mail page result")
	}
	if p.Complete && (p.NextCursor != "" || p.LimitReason != "") || !p.Complete && (p.NextCursor == "") == (p.LimitReason == "") || !validText(p.LimitReason, 64) {
		return errors.New("invalid mail completeness")
	}
	seen := map[string]bool{}
	for _, item := range p.Items {
		if err := item.Validate(); err != nil {
			return err
		}
		if seen[item.ID] {
			return errors.New("duplicate mail message")
		}
		seen[item.ID] = true
	}
	return nil
}

func (m Message) Validate(r ReadRequest) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := m.Summary.Validate(); err != nil {
		return err
	}
	if m.Summary.ID != r.MessageID || len(m.Body.Text) > r.BodyLimit() || !utf8.ValidString(m.Body.Text) || strings.ContainsFunc(m.Body.Text, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' }) || m.Body.OmittedReasons == nil || len(m.Body.OmittedReasons) > 16 || m.Body.Complete != (len(m.Body.OmittedReasons) == 0) {
		return errors.New("invalid mail body")
	}
	seen := map[string]bool{}
	for _, reason := range m.Body.OmittedReasons {
		if reason == "" || !validText(reason, 64) || seen[reason] {
			return errors.New("invalid mail omission")
		}
		seen[reason] = true
	}
	return nil
}
