package connector

import (
	"context"
	"encoding/json"
	"time"
)

const ContractVersion = "connector-contract-v4"

type OperationMode string

const (
	ModeCall           OperationMode = "call"
	ModeEnqueue        OperationMode = "enqueue"
	ModeStartOperation OperationMode = "start_operation"
)

// Connection contains only SecretRefs declared by the selected Provider
// descriptor. Runtime retains reference resolution and persistence ownership.
type Connection struct {
	Key          string            `json:"key"`
	WorkspaceID  string            `json:"workspace_id,omitempty"`
	ConnectorKey string            `json:"connector_key"`
	ProviderKey  string            `json:"provider_key"`
	Name         string            `json:"name,omitempty"`
	Status       string            `json:"status,omitempty"`
	Config       map[string]any    `json:"config,omitempty"`
	SecretRefs   map[string]string `json:"secret_refs,omitempty"`
	CreatedBy    string            `json:"created_by,omitempty"`
	CreatedAt    string            `json:"created_at,omitempty"`
	UpdatedAt    string            `json:"updated_at,omitempty"`
}

type Principal struct {
	UserID          string `json:"user_id,omitempty"`
	RoleKey         string `json:"role_key,omitempty"`
	DepartmentID    string `json:"department_id,omitempty"`
	RequestID       string `json:"request_id,omitempty"`
	CorrelationID   string `json:"correlation_id,omitempty"`
	CausationID     string `json:"causation_id,omitempty"`
	SurfaceKey      string `json:"surface_key,omitempty"`
	WorkspaceID     string `json:"workspace_id,omitempty"`
	IsAuthenticated bool   `json:"is_authenticated,omitempty"`
}

// CallRequest is the type-erased Runtime-to-provider envelope. Payload is
// decoded by a BoundOperation before project provider code is invoked. Secrets
// contains Runtime-resolved material only for fields declared by the selected
// Provider descriptor; Adapters must not resolve references.
type CallRequest struct {
	ConnectorKey   string            `json:"connector_key"`
	ProviderKey    string            `json:"provider_key"`
	OperationKey   string            `json:"operation_key"`
	ContractSHA256 string            `json:"contract_sha256"`
	Mode           OperationMode     `json:"mode"`
	Connection     Connection        `json:"connection"`
	Payload        json.RawMessage   `json:"payload"`
	RequestRef     string            `json:"request_ref,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Secrets        map[string]string `json:"secrets,omitempty"`
	Delivery       bool              `json:"delivery,omitempty"`
	Timeout        time.Duration     `json:"timeout,omitempty"`
	Principal      Principal         `json:"principal"`
}

type CallResult struct {
	Payload        json.RawMessage       `json:"payload,omitempty"`
	ResponseRef    string                `json:"response_ref,omitempty"`
	SecretUpdates  map[string]string     `json:"secret_updates,omitempty"`
	ResourceHealth *ResourceHealthReport `json:"resource_health,omitempty"`
}

// ResourceHealthReport is an optional, provider-normalized observation. A
// provider that lacks a reliable API or explicit provider error must omit it;
// Runtime never infers quota or billing state from a generic failed call.
type ResourceHealthReport struct {
	ObservationID     string `json:"observation_id"`
	Kind              string `json:"kind"`
	State             string `json:"state"`
	PreviousState     string `json:"previous_state,omitempty"`
	EvidenceSource    string `json:"evidence_source"`
	QuotaUsedPercent  *int   `json:"quota_used_percent,omitempty"`
	BalanceBand       string `json:"balance_band,omitempty"`
	CapabilityBlocked bool   `json:"capability_blocked"`
	ObservedAt        string `json:"observed_at"`
	ErrorCode         string `json:"error_code,omitempty"`
}

// Gateway is implemented by Runtime. Generated clients may call it only
// through the typed helpers in this package.
type Gateway interface {
	Call(context.Context, CallRequest) (CallResult, error)
}

// Adapter is the single Runtime-facing project Connector boundary. One
// Adapter may own many differently typed operations for one exact provider.
type Adapter interface {
	Descriptor() ProviderDescriptor
	Call(context.Context, CallRequest) (CallResult, error)
}

// ConfigValidator is optional. Runtime supplies the normalized connection and
// retains ownership of persistence and secret resolution; providers may only
// validate provider-specific configuration semantics.
type ConfigValidator interface {
	ValidateConfig(Connection) error
}

// TestConnectionRequest is a management-plane probe envelope. It deliberately
// has no operation key or payload: testing a configured connection is not a
// business operation and must not be disguised as Adapter.Call.
type TestConnectionRequest struct {
	ConnectorKey string            `json:"connector_key"`
	ProviderKey  string            `json:"provider_key"`
	Connection   Connection        `json:"connection"`
	Secrets      map[string]string `json:"secrets,omitempty"`
	Timeout      time.Duration     `json:"timeout,omitempty"`
	Principal    Principal         `json:"principal"`
}

type TestConnectionResult struct {
	Connected     bool              `json:"connected"`
	Details       json.RawMessage   `json:"details,omitempty"`
	SecretUpdates map[string]string `json:"secret_updates,omitempty"`
}

// ConnectionTester is optional. Runtime discovers it by type assertion and
// fails closed when a provider does not publish connection testing support.
type ConnectionTester interface {
	TestConnection(context.Context, TestConnectionRequest) (TestConnectionResult, error)
}

type VerifyWebhookRequest struct {
	ConnectorKey string              `json:"connector_key"`
	ProviderKey  string              `json:"provider_key"`
	Connection   Connection          `json:"connection"`
	Headers      map[string][]string `json:"headers,omitempty"`
	Query        map[string][]string `json:"query,omitempty"`
	Secrets      map[string]string   `json:"secrets,omitempty"`
	Body         []byte              `json:"body"`
	ReceivedAt   time.Time           `json:"received_at"`
}

type WebhookSecurityEvidence struct {
	SignatureVerified bool      `json:"signature_verified"`
	Nonce             string    `json:"nonce,omitempty"`
	DeviceIdentity    string    `json:"device_identity,omitempty"`
	EventTime         time.Time `json:"event_time,omitempty"`
}

type WebhookExternalIdentity struct {
	Subject     string `json:"subject"`
	SubjectType string `json:"subject_type,omitempty"`
	Name        string `json:"name,omitempty"`
	Group       string `json:"group,omitempty"`
}

type WebhookDeliveryReceipt struct {
	ResponseRef string    `json:"response_ref,omitempty"`
	Status      string    `json:"status,omitempty"`
	Error       string    `json:"error,omitempty"`
	OccurredAt  time.Time `json:"occurred_at,omitempty"`
}

type VerifiedWebhook struct {
	EventType        string                   `json:"event_type"`
	ExternalID       string                   `json:"external_id"`
	Payload          json.RawMessage          `json:"payload"`
	Security         *WebhookSecurityEvidence `json:"security,omitempty"`
	Challenge        string                   `json:"challenge,omitempty"`
	ChallengeFormat  string                   `json:"challenge_format,omitempty"`
	ExternalIdentity *WebhookExternalIdentity `json:"external_identity,omitempty"`
	DeliveryReceipt  *WebhookDeliveryReceipt  `json:"delivery_receipt,omitempty"`
}

// WebhookVerifier is optional. Runtime owns the HTTP route, request limits,
// replay/idempotency checks and persistence; providers only authenticate and
// normalize one already-resolved connection's inbound request.
type WebhookVerifier interface {
	VerifyWebhook(context.Context, VerifyWebhookRequest) (VerifiedWebhook, error)
}
