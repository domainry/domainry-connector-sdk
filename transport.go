package connector

import (
	"context"
	"time"
)

// Transport is the only outbound I/O capability supplied to project-owned
// Connector Adapters. Runtime owns the concrete clients, pools and limits.
type Transport interface {
	RoundTripHTTP(context.Context, HTTPRequest) (HTTPResponse, error)
	ExecuteSQL(context.Context, SQLRequest) (SQLResult, error)
}

// SMTPTransport is an optional Runtime-owned capability for Providers that
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

// MQTTTransport is an optional Runtime-owned capability for Providers that
// connect to MQTT brokers. Runtime owns sockets, TLS, packet framing,
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

// FilesystemTransport is an optional Runtime-owned capability for Providers
// that operate on a configured local filesystem. Runtime owns root and path
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

// ProcessTransport is an optional Runtime-owned capability for Providers that
// speak a line-delimited protocol over a governed child process. Runtime owns
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
	// SecretHeaders contains resolved secret-bearing header values that Runtime
	// injects immediately before dispatch. Keys must not occur in Headers.
	SecretHeaders map[string][]string `json:"-"`
	// SecretQuery is resolved secret material that Runtime injects immediately
	// before dispatch. It is deliberately excluded from serialization.
	SecretQuery map[string]string `json:"-"`
	// SecretForm contains resolved form fields that Runtime injects into an
	// application/x-www-form-urlencoded body immediately before dispatch.
	// Keys must not occur in the public Body.
	SecretForm map[string]string `json:"-"`
	// SecretJSON contains resolved string fields that Runtime injects into a
	// top-level application/json object immediately before dispatch. Keys must
	// not occur in the public Body.
	SecretJSON       map[string]string `json:"-"`
	Body             []byte            `json:"body,omitempty"`
	MaxResponseBytes int64             `json:"max_response_bytes,omitempty"`
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

// ProviderSetFactory binds project Adapter constructors to the Runtime-owned
// Transport before their descriptors enter the frozen Registry.
type ProviderSetFactory func(Transport) (ProviderSet, error)
