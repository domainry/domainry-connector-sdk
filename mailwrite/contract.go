// Package mailwrite defines explicit, bounded outgoing mail contracts. It owns
// no mailbox, credentials, transport, confirmation or durable execution state.
package mailwrite

import "github.com/domainry/domainry-connector-sdk/mail"

const (
	SendOperationKey  = "mail_send"
	ReplyOperationKey = "mail_reply"
)

// Message is the complete, user-visible outgoing content. Recipient arrays
// are explicit, including empty CC/BCC. From, arbitrary headers, raw MIME,
// attachments and provider idempotency keys cannot be supplied by a caller.
// Providers derive the sender and correlation from their trusted envelope.
type Message struct {
	To      []mail.Address `json:"to"`
	CC      []mail.Address `json:"cc"`
	BCC     []mail.Address `json:"bcc"`
	Subject string         `json:"subject"`
	Text    string         `json:"text"`
}

type SendRequest struct {
	Message Message `json:"message"`
}

// Reply requires a current read of the original message in the same account.
// The provider derives native thread references and RFC headers from that
// read; the caller supplies neither. To contains every original Reply-To
// recipient (or From when Reply-To is absent). Additional recipients must be
// listed explicitly; reply-all never silently appends hidden targets.
type ReplyRequest struct {
	MessageID string  `json:"message_id"`
	Message   Message `json:"message"`
}

// Result proves provider acceptance only. A Graph 202 has no message ID and
// must not invent one or claim recipient delivery. Integration persists this
// bounded receipt without mirroring the outgoing message body.
type Result struct {
	RequestRef         string `json:"request_ref"`
	Status             string `json:"status"` // accepted
	MessageID          string `json:"message_id,omitempty"`
	ThreadID           string `json:"thread_id,omitempty"`
	InReplyToMessageID string `json:"in_reply_to_message_id,omitempty"`
	AcceptedAt         string `json:"accepted_at"` // host-observed RFC3339 instant
	Delivery           string `json:"delivery"`    // unknown
}
