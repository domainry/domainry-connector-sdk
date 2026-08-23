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

type HTTPRequest struct {
	Method           string              `json:"method"`
	URL              string              `json:"url"`
	Headers          map[string][]string `json:"headers,omitempty"`
	Body             []byte              `json:"body,omitempty"`
	MaxResponseBytes int64               `json:"max_response_bytes,omitempty"`
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

// ExtensionFactory binds project Adapter constructors to the Runtime-owned
// Transport before their descriptors enter the frozen Registry.
type ExtensionFactory func(Transport) (ExtensionSet, error)
