// Package mail defines bounded, provider-neutral mailbox read contracts. Search
// syntax is explicitly provider-owned. This package owns no I/O or accounts.
package mail

type PageRequest struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// Query is native Gmail search or Microsoft Graph KQL, without URL encoding or
// Graph's outer double quotes. The provider must reject the other syntax.
type SearchRequest struct {
	Query       string `json:"query"`
	QuerySyntax string `json:"query_syntax"`
	Limit       int    `json:"limit,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
}

type ReadRequest struct {
	MessageID    string `json:"message_id"`
	MaxBodyBytes int    `json:"max_body_bytes,omitempty"`
}

type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

// Message IDs are opaque identifiers within one authorized account. RFC message
// IDs are references, not lookup keys. Missing dates and flags remain unknown.
type Summary struct {
	ID                string    `json:"id"`
	ThreadID          string    `json:"thread_id,omitempty"`
	InternetMessageID string    `json:"internet_message_id,omitempty"`
	Subject           string    `json:"subject,omitempty"`
	From              []Address `json:"from"`
	To                []Address `json:"to"`
	CC                []Address `json:"cc"`
	ReplyTo           []Address `json:"reply_to"`
	ReceivedAt        string    `json:"received_at,omitempty"`
	SentAt            string    `json:"sent_at,omitempty"`
	IsRead            *bool     `json:"is_read,omitempty"`
	IsDraft           *bool     `json:"is_draft,omitempty"`
	MetadataComplete  bool      `json:"metadata_complete"`
}

// Complete refers to the search/list traversal, never a mailbox snapshot or a
// count estimate. A provider cap has no next cursor and a nonempty LimitReason.
type MessagesPage struct {
	Items        []Summary `json:"items"`
	NextCursor   string    `json:"next_cursor,omitempty"`
	Complete     bool      `json:"complete"`
	LimitReason  string    `json:"limit_reason,omitempty"`
	QuerySyntax  string    `json:"query_syntax"`
	MailboxScope string    `json:"mailbox_scope"`
}

// Body contains extracted plain text only. Complete describes the primary
// textual body, not attachments, HTML presentation or external linked content.
// OmittedReasons are stable codes, never vendor error bodies or secrets.
type Body struct {
	Text           string   `json:"text"`
	Complete       bool     `json:"complete"`
	OmittedReasons []string `json:"omitted_reasons"`
}

type Message struct {
	Summary Summary `json:"summary"`
	Body    Body    `json:"body"`
}
