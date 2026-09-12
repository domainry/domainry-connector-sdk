package mail_test

import (
	"strings"
	"testing"

	"github.com/domainry/domainry-connector-sdk/mail"
)

func TestContractIdentity(t *testing.T) {
	if got := mail.ComputedContractSHA256(); got != mail.ContractSHA256 {
		t.Fatalf("contract hash: %s", got)
	}
	seen := map[string]bool{}
	for _, key := range []string{mail.ListOperationKey, mail.SearchOperationKey, mail.ReadOperationKey} {
		h := mail.OperationSHA256(key)
		if len(h) != 64 || seen[h] {
			t.Fatal(key, h)
		}
		seen[h] = true
	}
	if mail.OperationSHA256("mail_send") != "" {
		t.Fatal("write acquired read identity")
	}
	for key, want := range map[string]string{
		mail.ListOperationKey:   "a67a8c1553a2b8457dd54dd3d574994f05e8177ccae2b34a78239affa3fd3004",
		mail.SearchOperationKey: "ca712330f5266cceb7fb4f46bbcca13f836e22c60bcc70d3fe95845e5818575f",
		mail.ReadOperationKey:   "e98aba01bcab28ef70dd6dd5cac5c45a64e2d6d31899dc2e3105bb7a63f57a23",
	} {
		if got := mail.OperationSHA256(key); got != want {
			t.Fatal(key, got, want)
		}
	}
}

func summary() mail.Summary {
	return mail.Summary{ID: "opaque/a", From: []mail.Address{}, To: []mail.Address{}, CC: []mail.Address{}, ReplyTo: []mail.Address{}, MetadataComplete: true}
}

func TestMailReadContractDoesNotHidePartialOrUnknownResults(t *testing.T) {
	page := mail.MessagesPage{Items: []mail.Summary{summary()}, QuerySyntax: mail.GmailSyntax, MailboxScope: "mailbox_excluding_spam_trash", Complete: true}
	if err := page.Validate(10); err != nil {
		t.Fatal(err)
	}
	page.Complete = false
	if page.Validate(10) == nil {
		t.Fatal("unknown end treated as complete")
	}
	page.LimitReason = "provider_search_limit"
	if err := page.Validate(10); err != nil {
		t.Fatal(err)
	}
	page.NextCursor = "next"
	if page.Validate(10) == nil {
		t.Fatal("cap offered continuation")
	}
	page.LimitReason = ""
	if err := page.Validate(10); err != nil {
		t.Fatal(err)
	}
	page.Items = append(page.Items, summary())
	if page.Validate(10) == nil {
		t.Fatal("duplicate results accepted")
	}

	r := mail.ReadRequest{MessageID: "opaque/a", MaxBodyBytes: 1024}
	m := mail.Message{Summary: summary(), Body: mail.Body{Text: "正文\n内容", Complete: true, OmittedReasons: []string{}}}
	if err := m.Validate(r); err != nil {
		t.Fatal(err)
	}
	m.Body.Text = strings.Repeat("你", 342)
	if m.Validate(r) == nil {
		t.Fatal("body bound counted characters instead of bytes")
	}
	m.Body = mail.Body{Text: "partial", Complete: true, OmittedReasons: []string{"body_truncated"}}
	if m.Validate(r) == nil {
		t.Fatal("truncation declared complete")
	}
	m.Body.Complete = false
	if err := m.Validate(r); err != nil {
		t.Fatal(err)
	}
	m.Summary.ID = "different"
	if m.Validate(r) == nil {
		t.Fatal("wrong message disclosed")
	}
}

func TestMailInputsAreBoundedAndDatesRemainInstants(t *testing.T) {
	for _, id := range []string{".", "..", "mail/../other"} {
		if mail.ValidID(id) {
			t.Fatal("path segment accepted as mail ID", id)
		}
	}
	for _, p := range []mail.PageRequest{{Limit: -1}, {Limit: 26}, {Cursor: strings.Repeat("x", 8193)}, {Cursor: "token\n"}} {
		if p.Validate() == nil {
			t.Fatal(p)
		}
	}
	for _, s := range []mail.SearchRequest{{Query: "", QuerySyntax: mail.GmailSyntax}, {Query: "x", QuerySyntax: "sql"}, {Query: "x\n", QuerySyntax: mail.GraphSyntax}} {
		if s.Validate() == nil {
			t.Fatal(s)
		}
	}
	for _, r := range []mail.ReadRequest{{MessageID: ""}, {MessageID: "a", MaxBodyBytes: 1023}, {MessageID: "a", MaxBodyBytes: 65537}} {
		if r.Validate() == nil {
			t.Fatal(r)
		}
	}
	s := summary()
	s.ReceivedAt = "2026-11-01T01:30:00-04:00"
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.ReceivedAt = "2026-11-01T01:30:00"
	if s.Validate() == nil {
		t.Fatal("offset-free mail date accepted")
	}
}
