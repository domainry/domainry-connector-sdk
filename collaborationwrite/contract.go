// Package collaborationwrite defines bounded, synchronous collaboration
// message writes. Credentials, account authority, confirmation and request
// identity are owned by the host and never appear in the payload.
package collaborationwrite

const SendOperationKey = "collaboration_message_send"

type SendRequest struct {
	Recipient string `json:"recipient"`
	Text      string `json:"text"`
}

// Result acknowledges provider acceptance, not human delivery or reading.
type Result struct {
	RequestRef string `json:"request_ref"`
	Status     string `json:"status"`
	MessageID  string `json:"message_id"`
	AcceptedAt string `json:"accepted_at"`
	Delivery   string `json:"delivery"`
}
