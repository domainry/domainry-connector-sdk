package mailwrite_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/domainry/domainry-connector-sdk/mail"
	write "github.com/domainry/domainry-connector-sdk/mailwrite"
)

func message() write.Message {
	return write.Message{To: []mail.Address{{Address: "sender@example.test", Name: "原发件人"}}, CC: []mail.Address{}, BCC: []mail.Address{}, Subject: "项目评审", Text: "时间已确认。\n谢谢。"}
}

func original() mail.Summary {
	no := false
	return mail.Summary{ID: "original-message", Subject: "项目评审", From: []mail.Address{{Address: "sender@example.test"}}, To: []mail.Address{{Address: "me@example.test"}}, CC: []mail.Address{}, ReplyTo: []mail.Address{}, IsDraft: &no, MetadataComplete: true}
}

func TestContractIdentityKeepsReadAndWriteOperationsDistinct(t *testing.T) {
	if got := write.ComputedContractSHA256(); got != write.ContractSHA256 {
		t.Fatalf("mail write contract SHA256=%s", got)
	}
	first, second := write.OperationSHA256(write.SendOperationKey), write.OperationSHA256(write.ReplyOperationKey)
	if len(first) != 64 || len(second) != 64 || first == second || mail.OperationSHA256(write.SendOperationKey) != "" || mail.OperationSHA256(write.ReplyOperationKey) != "" || write.OperationSHA256(mail.ReadOperationKey) != "" || write.OperationSHA256("mail_delete") != "" || mail.ComputedContractSHA256() != mail.ContractSHA256 {
		t.Fatal("mail write changed read operations or declared a foreign operation")
	}
}

func TestOutgoingMailUsesOnlyExplicitValidBoundedRecipientsAndText(t *testing.T) {
	if err := message().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*write.Message){
		"unknown-to":              func(m *write.Message) { m.To = nil },
		"unknown-cc":              func(m *write.Message) { m.CC = nil },
		"unknown-bcc":             func(m *write.Message) { m.BCC = nil },
		"no-recipients":           func(m *write.Message) { m.To = []mail.Address{} },
		"header-injection":        func(m *write.Message) { m.To[0].Address = "sender@example.test\r\nBcc: hidden@example.test" },
		"name-injection":          func(m *write.Message) { m.To[0].Name = "Person\nBcc: hidden" },
		"multiple-addresses":      func(m *write.Message) { m.To[0].Address = "one@example.test,two@example.test" },
		"display-in-address":      func(m *write.Message) { m.To[0].Address = "Person <sender@example.test>" },
		"duplicate-across-groups": func(m *write.Message) { m.BCC = []mail.Address{{Address: "SENDER@example.test"}} },
		"subject-injection":       func(m *write.Message) { m.Subject = "subject\r\nTo: other@example.test" },
		"subject-limit":           func(m *write.Message) { m.Subject = strings.Repeat("x", write.MaximumSubjectBytes+1) },
		"body-limit":              func(m *write.Message) { m.Text = strings.Repeat("x", write.MaximumTextBytes+1) },
		"body-control":            func(m *write.Message) { m.Text = "body\x00data" },
		"body-invalid-utf8":       func(m *write.Message) { m.Text = string([]byte{0xff}) },
		"empty-message":           func(m *write.Message) { m.Subject, m.Text = " ", "\n" },
	} {
		t.Run(name, func(t *testing.T) {
			m := message()
			change(&m)
			if m.Validate() == nil {
				t.Fatal("invalid outgoing message accepted")
			}
		})
	}
	m := message()
	m.To, m.BCC = []mail.Address{}, m.To
	if err := m.Validate(); err != nil {
		t.Fatal("explicit BCC-only mail rejected", err)
	}
	m.To = []mail.Address{}
	for i := 0; i < write.MaximumRecipients; i++ {
		m.To = append(m.To, mail.Address{Address: fmt.Sprintf("person%d@example.test", i)})
	}
	if m.Validate() == nil {
		t.Fatal("recipient limit ignored BCC")
	}
	m.BCC = []mail.Address{}
	if err := m.Validate(); err != nil {
		t.Fatal("exact total recipient limit rejected", err)
	}
}

func TestReplyUsesCurrentCompleteMetadataAndReplyTo(t *testing.T) {
	r := write.ReplyRequest{MessageID: "original-message", Message: message()}
	r.Message.Subject = "Re: 项目评审"
	if err := r.ValidateAgainst(original()); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*mail.Summary){
		"another-message":         func(s *mail.Summary) { s.ID = "other" },
		"incomplete-metadata":     func(s *mail.Summary) { s.MetadataComplete = false },
		"draft":                   func(s *mail.Summary) { yes := true; s.IsDraft = &yes },
		"unknown-draft-state":     func(s *mail.Summary) { s.IsDraft = nil },
		"changed-subject":         func(s *mail.Summary) { s.Subject = "different" },
		"unknown-sender":          func(s *mail.Summary) { s.From = []mail.Address{} },
		"reply-to-overrides-from": func(s *mail.Summary) { s.ReplyTo = []mail.Address{{Address: "reply@example.test"}} },
	} {
		t.Run(name, func(t *testing.T) {
			s := original()
			change(&s)
			if r.ValidateAgainst(s) == nil {
				t.Fatal("reply guessed missing or different source data")
			}
		})
	}
	s := original()
	s.ReplyTo = []mail.Address{{Address: "reply@example.test"}}
	r.Message.To = []mail.Address{{Address: "reply@example.test"}, {Address: "approved-extra@example.test"}}
	if err := r.ValidateAgainst(s); err != nil {
		t.Fatal("explicit reply targets rejected", err)
	}
	r.Message.To = []mail.Address{{Address: "approved-extra@example.test"}}
	r.Message.BCC = []mail.Address{{Address: "reply@example.test"}}
	if r.ValidateAgainst(s) == nil {
		t.Fatal("original reply target silently moved to hidden recipients")
	}
}

func TestOutgoingJSONRejectsHiddenSenderHeadersAndNativePayload(t *testing.T) {
	r := write.SendRequest{Message: message()}
	r.Message.Text = strings.Repeat("<", write.MaximumTextBytes)
	raw, _ := json.Marshal(r)
	var got write.SendRequest
	if err := json.Unmarshal(raw, &got); err != nil || got.Message.Text != r.Message.Text {
		t.Fatal("valid escaped text failed its bounded round trip", err)
	}
	for _, field := range []string{`"from":"other@example.test"`, `"headers":{"Bcc":"other@example.test"}`, `"raw":"MIME"`, `"access_token":"secret"`, `"request_ref":"model-key"`, `"attachments":[]`} {
		modified := strings.Replace(string(raw), `"message":{`, `"message":{`+field+`,`, 1)
		if json.Unmarshal([]byte(modified), &got) == nil {
			t.Fatal("undeclared outgoing field accepted", field)
		}
	}
	for _, input := range []string{"null", "{}", strings.Repeat(" ", write.MaximumRequestBytes+1)} {
		if json.Unmarshal([]byte(input), &got) == nil {
			t.Fatal("invalid request object accepted")
		}
	}
	reply := write.ReplyRequest{MessageID: "original-message", Message: message()}
	raw, _ = json.Marshal(reply)
	var decoded write.ReplyRequest
	if err := json.Unmarshal(raw, &decoded); err != nil || decoded.MessageID != reply.MessageID {
		t.Fatal("reply round trip", err)
	}
	modified := strings.Replace(string(raw), `"message_id":"original-message"`, `"message_id":"original-message","thread_id":"model-thread"`, 1)
	if json.Unmarshal([]byte(modified), &decoded) == nil {
		t.Fatal("model supplied a native thread identifier")
	}
}

func TestReceiptDistinguishesAcceptanceFromDeliveryAndBindsOriginalRequest(t *testing.T) {
	r := write.Result{RequestRef: "host-request", Status: "accepted", AcceptedAt: "2026-09-12T08:00:00+08:00", Delivery: "unknown"}
	if err := r.Validate(write.SendOperationKey, "host-request", ""); err != nil {
		t.Fatal("Graph 202 without a message ID rejected", err)
	}
	r.MessageID, r.ThreadID = "actual-provider-message", "actual-thread"
	if err := r.Validate(write.SendOperationKey, "host-request", ""); err != nil {
		t.Fatal("Gmail acceptance receipt rejected", err)
	}
	for name, change := range map[string]func(*write.Result){
		"wrong-request":           func(r *write.Result) { r.RequestRef = "other" },
		"claimed-delivery":        func(r *write.Result) { r.Delivery = "delivered" },
		"claimed-completion":      func(r *write.Result) { r.Status = "completed" },
		"missing-time":            func(r *write.Result) { r.AcceptedAt = "" },
		"ambiguous-time":          func(r *write.Result) { r.AcceptedAt = "2026-09-12T08:00:00" },
		"thread-without-message":  func(r *write.Result) { r.MessageID = "" },
		"new-mail-claiming-reply": func(r *write.Result) { r.InReplyToMessageID = "original-message" },
	} {
		t.Run(name, func(t *testing.T) {
			copy := r
			change(&copy)
			if copy.Validate(write.SendOperationKey, "host-request", "") == nil {
				t.Fatal("invalid mail receipt accepted")
			}
		})
	}
	r.InReplyToMessageID = "original-message"
	if r.Validate(write.ReplyOperationKey, "host-request", "original-message") != nil || r.Validate(write.ReplyOperationKey, "host-request", "different-message") == nil {
		t.Fatal("reply receipt failed exact original-message binding")
	}
}
