package connector

import (
	"context"
	"fmt"
	"io"
	"time"
	"unicode/utf8"
)

// Transport is the only outbound I/O capability supplied to project-owned
// Connector Adapters. The deployment host owns the concrete clients, pools and limits.
type Transport interface {
	RoundTripHTTP(context.Context, HTTPRequest) (HTTPResponse, error)
	ExecuteSQL(context.Context, SQLRequest) (SQLResult, error)
}

// SMTPTransport is an optional host-owned capability for Providers that
// deliver RFC 5322 messages. Keeping it separate preserves compatibility for
// transports that only support HTTP and SQL.
type SMTPTransport interface {
	SendSMTP(context.Context, SMTPRequest) (SMTPResult, error)
}

type SMTPRequest struct {
	Host           string        `json:"host"`
	Port           int           `json:"port"`
	ImplicitTLS    bool          `json:"implicit_tls,omitempty"`
	StartTLS       bool          `json:"start_tls,omitempty"`
	TLSServerName  string        `json:"tls_server_name,omitempty"`
	TLSCAPEM       string        `json:"tls_ca_pem,omitempty"`
	Username       string        `json:"username,omitempty"`
	SecretPassword string        `json:"-"`
	EnvelopeFrom   string        `json:"envelope_from,omitempty"`
	Recipients     []string      `json:"recipients,omitempty"`
	Message        []byte        `json:"message,omitempty"`
	ProbeOnly      bool          `json:"probe_only,omitempty"`
	Timeout        time.Duration `json:"timeout,omitempty"`
}

type SMTPResult struct {
	Connected bool `json:"connected"`
	Accepted  bool `json:"accepted"`
}

// MQTTTransport is an optional host-owned capability for Providers that
// connect to MQTT brokers. The host owns sockets, TLS, packet framing,
// deadlines, QoS acknowledgement, and secret injection.
type MQTTTransport interface {
	ExecuteMQTT(context.Context, MQTTRequest) (MQTTResult, error)
}

type MQTTRequest struct {
	BrokerURL         string        `json:"broker_url"`
	ClientID          string        `json:"client_id,omitempty"`
	Topic             string        `json:"topic,omitempty"`
	Payload           []byte        `json:"payload,omitempty"`
	QoS               byte          `json:"qos,omitempty"`
	ProbeOnly         bool          `json:"probe_only,omitempty"`
	Timeout           time.Duration `json:"timeout,omitempty"`
	TLSServerName     string        `json:"tls_server_name,omitempty"`
	TLSCAPEM          string        `json:"tls_ca_pem,omitempty"`
	SecretUsername    string        `json:"-"`
	SecretPassword    string        `json:"-"`
	SecretCertificate string        `json:"-"`
	SecretPrivateKey  string        `json:"-"`
}

type MQTTResult struct {
	Connected bool   `json:"connected"`
	Accepted  bool   `json:"accepted,omitempty"`
	PacketID  uint16 `json:"packet_id,omitempty"`
}

// WebSocketTransport is an optional host-owned capability for Providers that
// consume or publish framed WebSocket messages. The host owns dialing, TLS,
// redirects, frame limits, deadlines, secret header injection, and closure.
// The context passed to OpenWebSocket owns the returned session: cancellation
// forcibly closes the connection even when a Provider misses an explicit
// Close path. Providers should still call Close to attempt a normal close
// handshake before their owner context is cancelled.
type WebSocketTransport interface {
	OpenWebSocket(context.Context, WebSocketRequest) (WebSocketSession, error)
}

type WebSocketMessageType string

const (
	WebSocketMessageText   WebSocketMessageType = "text"
	WebSocketMessageBinary WebSocketMessageType = "binary"
)

type WebSocketRequest struct {
	URL             string              `json:"url"`
	Headers         map[string][]string `json:"headers,omitempty"`
	SecretHeaders   map[string][]string `json:"-"`
	Subprotocols    []string            `json:"subprotocols,omitempty"`
	MaxMessageBytes int64               `json:"max_message_bytes,omitempty"`
}

type WebSocketMessage struct {
	Type WebSocketMessageType `json:"type"`
	Data []byte               `json:"data"`
}

// WebSocketClose describes a protocol-owned close handshake. Code must be a
// wire-valid RFC 6455 or application status. A Provider chooses whether a
// remote session should be invalidated or remain resumable; the host validates
// and performs the bounded handshake.
type WebSocketClose struct {
	Code   int    `json:"code"`
	Reason string `json:"reason,omitempty"`
}

func (c WebSocketClose) Validate() error {
	if !validWebSocketCloseCode(c.Code) {
		return fmt.Errorf("WebSocket close code %d is not wire-valid", c.Code)
	}
	if !utf8.ValidString(c.Reason) || len(c.Reason) > 123 {
		return fmt.Errorf("WebSocket close reason must be valid UTF-8 up to 123 bytes")
	}
	return nil
}

func validWebSocketCloseCode(code int) bool {
	switch code {
	case 1004, 1005, 1006, 1015:
		return false
	}
	return (code >= 1000 && code <= 1014) || (code >= 3000 && code <= 4999)
}

type WebSocketSession interface {
	Send(context.Context, WebSocketMessage) error
	Receive(context.Context) (WebSocketMessage, error)
	Close(context.Context, WebSocketClose) error
}

// FilesystemTransport is an optional host-owned capability for Providers
// that operate on a configured local filesystem. The host owns root and path
// validation, permissions, bounded reads, durable staged writes, and deletion.
type FilesystemTransport interface {
	ExecuteFilesystem(context.Context, FilesystemRequest) (FilesystemResult, error)
}

type FilesystemOperation string

const (
	FilesystemOperationProbe  FilesystemOperation = "probe"
	FilesystemOperationRead   FilesystemOperation = "read"
	FilesystemOperationWrite  FilesystemOperation = "write"
	FilesystemOperationDelete FilesystemOperation = "delete"
)

type FilesystemRequest struct {
	Root         string              `json:"root"`
	Path         string              `json:"path,omitempty"`
	Operation    FilesystemOperation `json:"operation"`
	Content      []byte              `json:"content,omitempty"`
	MaxReadBytes int64               `json:"max_read_bytes,omitempty"`
}

type FilesystemResult struct {
	Path    string `json:"path,omitempty"`
	Content []byte `json:"content,omitempty"`
	Size    int64  `json:"size,omitempty"`
	Exists  bool   `json:"exists"`
}

// ProcessTransport is an optional host-owned capability for Providers that
// speak a line-delimited protocol over a governed child process. The host owns
// executable authorization, process creation, stdio, limits, and termination.
type ProcessTransport interface {
	StartProcess(context.Context, ProcessRequest) (ProcessSession, error)
}

type ProcessRequest struct {
	Executable       string        `json:"executable"`
	Arguments        []string      `json:"arguments,omitempty"`
	WorkingDirectory string        `json:"working_directory,omitempty"`
	MaxMessageBytes  int64         `json:"max_message_bytes,omitempty"`
	ShutdownGrace    time.Duration `json:"shutdown_grace,omitempty"`
}

type ProcessSession interface {
	SendLine(context.Context, []byte) error
	ReceiveLine(context.Context) ([]byte, error)
	Close(context.Context) error
}

type HTTPRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers,omitempty"`
	// SecretHeaders contains resolved secret-bearing header values that Integration
	// injects immediately before dispatch. Keys must not occur in Headers.
	SecretHeaders map[string][]string `json:"-"`
	// SecretQuery is resolved secret material that Integration injects immediately
	// before dispatch. It is deliberately excluded from serialization.
	SecretQuery map[string]string `json:"-"`
	// SecretForm contains resolved form fields that Integration injects into an
	// application/x-www-form-urlencoded body immediately before dispatch.
	// Keys must not occur in the public Body.
	SecretForm map[string]string `json:"-"`
	// SecretJSON contains resolved string fields that Integration injects into a
	// top-level application/json object immediately before dispatch. Keys must
	// not occur in the public Body.
	SecretJSON       map[string]string `json:"-"`
	Body             []byte            `json:"body,omitempty"`
	Content          io.Reader         `json:"-"`
	ContentLength    int64             `json:"-"`
	ContentSHA256    string            `json:"-"`
	MaxResponseBytes int64             `json:"max_response_bytes,omitempty"`
	Timeout          time.Duration     `json:"timeout,omitempty"`
}

type HTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Body       []byte              `json:"body,omitempty"`
}

type SQLOperation string

const (
	SQLOperationPing  SQLOperation = "ping"
	SQLOperationQuery SQLOperation = "query"
	SQLOperationExec  SQLOperation = "exec"
)

type SQLRequest struct {
	Driver    string        `json:"driver"`
	DSN       string        `json:"-"`
	Operation SQLOperation  `json:"operation"`
	Statement string        `json:"statement,omitempty"`
	Arguments []any         `json:"arguments,omitempty"`
	MaxRows   int           `json:"max_rows,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
}

type SQLResult struct {
	Columns      []string `json:"columns,omitempty"`
	Rows         [][]any  `json:"rows,omitempty"`
	RowsAffected int64    `json:"rows_affected,omitempty"`
	LastInsertID *int64   `json:"last_insert_id,omitempty"`
	Truncated    bool     `json:"truncated,omitempty"`
}

// ProviderSetFactory binds project Adapter constructors to the host-owned
// Transport before their descriptors enter the frozen Registry.
type ProviderSetFactory func(Transport) (ProviderSet, error)
