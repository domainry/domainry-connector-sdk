package connectorext

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type OperationEffect string

const (
	EffectRead    OperationEffect = "read"
	EffectReserve OperationEffect = "reserve"
	EffectWrite   OperationEffect = "write"
)

type IdempotencyStrategy string

const (
	IdempotencyNone        IdempotencyStrategy = "none"
	IdempotencyNatural     IdempotencyStrategy = "natural"
	IdempotencyProviderKey IdempotencyStrategy = "provider_key"
)

type IdempotencyContract struct {
	Strategy            IdempotencyStrategy `json:"strategy"`
	KeyRetentionSeconds int                 `json:"key_retention_seconds,omitempty"`
}

type ReconciliationMode string

const (
	ReconciliationNone           ReconciliationMode = "none"
	ReconciliationProviderLookup ReconciliationMode = "provider_lookup"
)

type CompensationMode string

const (
	CompensationNone     CompensationMode = "none"
	CompensationExplicit CompensationMode = "explicit"
	CompensationSaga     CompensationMode = "saga"
)

type CompensationContract struct {
	Mode         CompensationMode `json:"mode"`
	OperationKey string           `json:"operation_key,omitempty"`
}

type ReliabilityContract struct {
	Effect         OperationEffect      `json:"effect"`
	Idempotency    IdempotencyContract  `json:"idempotency"`
	Reconciliation ReconciliationMode   `json:"reconciliation"`
	Compensation   CompensationContract `json:"compensation"`
}

func (c ReliabilityContract) Validate(operationKey string) error {
	operationKey = strings.TrimSpace(operationKey)
	switch c.Effect {
	case EffectRead, EffectReserve, EffectWrite:
	default:
		return fmt.Errorf("connector operation %s has invalid effect %q", operationKey, c.Effect)
	}
	switch c.Idempotency.Strategy {
	case IdempotencyNone:
		if c.Idempotency.KeyRetentionSeconds != 0 {
			return fmt.Errorf("connector operation %s idempotency none cannot declare key retention", operationKey)
		}
	case IdempotencyNatural:
		if c.Idempotency.KeyRetentionSeconds != 0 {
			return fmt.Errorf("connector operation %s natural idempotency cannot declare key retention", operationKey)
		}
	case IdempotencyProviderKey:
		if c.Idempotency.KeyRetentionSeconds <= 0 {
			return fmt.Errorf("connector operation %s provider-key idempotency requires positive key retention", operationKey)
		}
	default:
		return fmt.Errorf("connector operation %s has invalid idempotency strategy %q", operationKey, c.Idempotency.Strategy)
	}
	if c.Effect == EffectRead && c.Idempotency.Strategy != IdempotencyNatural {
		return fmt.Errorf("connector read operation %s must declare natural idempotency", operationKey)
	}
	switch c.Reconciliation {
	case ReconciliationNone:
	case ReconciliationProviderLookup:
		if c.Effect == EffectRead {
			return fmt.Errorf("connector read operation %s cannot require provider reconciliation", operationKey)
		}
	default:
		return fmt.Errorf("connector operation %s has invalid reconciliation mode %q", operationKey, c.Reconciliation)
	}
	compensationKey := strings.TrimSpace(c.Compensation.OperationKey)
	switch c.Compensation.Mode {
	case CompensationNone:
		if compensationKey != "" {
			return fmt.Errorf("connector operation %s compensation none cannot reference an operation", operationKey)
		}
	case CompensationExplicit, CompensationSaga:
		if c.Effect == EffectRead {
			return fmt.Errorf("connector read operation %s cannot declare compensation", operationKey)
		}
		if compensationKey == "" || compensationKey == operationKey {
			return fmt.Errorf("connector operation %s requires a distinct compensation operation", operationKey)
		}
	default:
		return fmt.Errorf("connector operation %s has invalid compensation mode %q", operationKey, c.Compensation.Mode)
	}
	return nil
}

type ReconciliationOutcome string

const (
	ReconciliationSucceeded ReconciliationOutcome = "succeeded"
	ReconciliationFailed    ReconciliationOutcome = "failed"
	ReconciliationPending   ReconciliationOutcome = "pending"
	ReconciliationNotFound  ReconciliationOutcome = "not_found"
	ReconciliationUnknown   ReconciliationOutcome = "unknown"
)

type ReconcileRequest struct {
	ConnectorKey   string            `json:"connector_key"`
	ProviderKey    string            `json:"provider_key"`
	OperationKey   string            `json:"operation_key"`
	ContractSHA256 string            `json:"contract_sha256"`
	Connection     Connection        `json:"connection"`
	RequestRef     string            `json:"request_ref"`
	ResponseRef    string            `json:"response_ref,omitempty"`
	Payload        json.RawMessage   `json:"payload"`
	Secrets        map[string]string `json:"secrets,omitempty"`
	Timeout        time.Duration     `json:"timeout,omitempty"`
	Principal      Principal         `json:"principal"`
}

func (r ReconcileRequest) Validate() error {
	if strings.TrimSpace(r.ConnectorKey) == "" || r.ConnectorKey != strings.TrimSpace(r.ConnectorKey) || strings.TrimSpace(r.ProviderKey) == "" || r.ProviderKey != strings.TrimSpace(r.ProviderKey) {
		return fmt.Errorf("connector reconciliation requires canonical connector and provider keys")
	}
	if strings.TrimSpace(r.OperationKey) == "" || r.OperationKey != strings.TrimSpace(r.OperationKey) {
		return fmt.Errorf("connector reconciliation requires a canonical operation key")
	}
	if !operationContractSHA256Pattern.MatchString(r.ContractSHA256) {
		return fmt.Errorf("connector reconciliation operation contract SHA-256 is invalid")
	}
	if strings.TrimSpace(r.Connection.ConnectorKey) != r.ConnectorKey || strings.TrimSpace(r.Connection.ProviderKey) != r.ProviderKey {
		return fmt.Errorf("connector reconciliation connection does not match provider identity")
	}
	if strings.TrimSpace(r.RequestRef) == "" || r.RequestRef != strings.TrimSpace(r.RequestRef) {
		return fmt.Errorf("connector reconciliation requires a canonical request reference")
	}
	if len(r.Payload) == 0 || !json.Valid(r.Payload) {
		return fmt.Errorf("connector reconciliation payload must contain one JSON value")
	}
	if r.Timeout < 0 {
		return fmt.Errorf("connector reconciliation timeout cannot be negative")
	}
	return nil
}

type ReconcileResult struct {
	Outcome     ReconciliationOutcome `json:"outcome"`
	Result      *CallResult           `json:"result,omitempty"`
	FailureCode string                `json:"failure_code,omitempty"`
	RetryAfter  time.Duration         `json:"retry_after,omitempty"`
}

func (r ReconcileResult) Validate() error {
	switch r.Outcome {
	case ReconciliationSucceeded:
		if strings.TrimSpace(r.FailureCode) != "" || r.RetryAfter != 0 {
			return errorsForReconcileResult(r.Outcome)
		}
	case ReconciliationFailed:
		if !providerErrorCodePattern.MatchString(r.FailureCode) || r.RetryAfter != 0 {
			return errorsForReconcileResult(r.Outcome)
		}
	case ReconciliationPending:
		if strings.TrimSpace(r.FailureCode) != "" || r.RetryAfter <= 0 {
			return errorsForReconcileResult(r.Outcome)
		}
	case ReconciliationNotFound, ReconciliationUnknown:
		if strings.TrimSpace(r.FailureCode) != "" || r.RetryAfter != 0 || r.Result != nil {
			return errorsForReconcileResult(r.Outcome)
		}
	default:
		return fmt.Errorf("connector reconciliation outcome %q is invalid", r.Outcome)
	}
	return nil
}

func errorsForReconcileResult(outcome ReconciliationOutcome) error {
	return fmt.Errorf("connector reconciliation result for %q has incompatible fields", outcome)
}

// Reconciler is optional and read-only. Runtime invokes it only for an
// operation whose ReliabilityContract declares provider_lookup.
type Reconciler interface {
	Reconcile(context.Context, ReconcileRequest) (ReconcileResult, error)
}
