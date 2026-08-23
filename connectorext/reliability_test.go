package connectorext

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func reliabilityDescriptor(key string, mode OperationMode, reliability ReliabilityContract) OperationDescriptor {
	return OperationDescriptor{
		ConnectorKey: "payments", ProviderKey: "acme", Key: key, Mode: mode,
		ContractSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Reliability:    reliability,
	}
}

func TestReliabilityContractRejectsAmbiguousRecoverySemantics(t *testing.T) {
	validWrite := ReliabilityContract{
		Effect: EffectWrite, Idempotency: IdempotencyContract{Strategy: IdempotencyProviderKey, KeyRetentionSeconds: 3600},
		Reconciliation: ReconciliationProviderLookup, Compensation: CompensationContract{Mode: CompensationNone},
	}
	tests := []struct {
		name   string
		mutate func(*ReliabilityContract)
		want   string
	}{
		{name: "effect", mutate: func(value *ReliabilityContract) { value.Effect = "unknown" }, want: "invalid effect"},
		{name: "idempotency", mutate: func(value *ReliabilityContract) { value.Idempotency.Strategy = "best_effort" }, want: "invalid idempotency"},
		{name: "provider key retention", mutate: func(value *ReliabilityContract) { value.Idempotency.KeyRetentionSeconds = 0 }, want: "positive key retention"},
		{name: "none retention", mutate: func(value *ReliabilityContract) {
			value.Idempotency = IdempotencyContract{Strategy: IdempotencyNone, KeyRetentionSeconds: 1}
		}, want: "cannot declare key retention"},
		{name: "natural retention", mutate: func(value *ReliabilityContract) {
			value.Idempotency = IdempotencyContract{Strategy: IdempotencyNatural, KeyRetentionSeconds: 1}
		}, want: "cannot declare key retention"},
		{name: "reconciliation", mutate: func(value *ReliabilityContract) { value.Reconciliation = "guess" }, want: "invalid reconciliation"},
		{name: "compensation mode", mutate: func(value *ReliabilityContract) { value.Compensation.Mode = "automatic" }, want: "invalid compensation"},
		{name: "missing compensation", mutate: func(value *ReliabilityContract) { value.Compensation = CompensationContract{Mode: CompensationSaga} }, want: "distinct compensation"},
		{name: "self compensation", mutate: func(value *ReliabilityContract) {
			value.Compensation = CompensationContract{Mode: CompensationSaga, OperationKey: "charge"}
		}, want: "distinct compensation"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validWrite
			test.mutate(&value)
			if err := value.Validate("charge"); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	invalidRead := readReliability
	invalidRead.Idempotency.Strategy = IdempotencyNone
	if err := invalidRead.Validate("lookup"); err == nil || !strings.Contains(err.Error(), "natural idempotency") {
		t.Fatalf("read idempotency error=%v", err)
	}
	invalidRead = readReliability
	invalidRead.Reconciliation = ReconciliationProviderLookup
	if err := invalidRead.Validate("lookup"); err == nil || !strings.Contains(err.Error(), "cannot require provider reconciliation") {
		t.Fatalf("read reconciliation error=%v", err)
	}
	invalidRead = readReliability
	invalidRead.Compensation = CompensationContract{Mode: CompensationExplicit, OperationKey: "undo"}
	if err := invalidRead.Validate("lookup"); err == nil || !strings.Contains(err.Error(), "cannot declare compensation") {
		t.Fatalf("read compensation error=%v", err)
	}
}

func TestProviderCompensationMustReferenceOneTerminalEnqueueWrite(t *testing.T) {
	reserve := ReliabilityContract{
		Effect: EffectReserve, Idempotency: IdempotencyContract{Strategy: IdempotencyProviderKey, KeyRetentionSeconds: 3600},
		Reconciliation: ReconciliationProviderLookup, Compensation: CompensationContract{Mode: CompensationSaga, OperationKey: "release"},
	}
	release := ReliabilityContract{
		Effect: EffectWrite, Idempotency: IdempotencyContract{Strategy: IdempotencyProviderKey, KeyRetentionSeconds: 3600},
		Reconciliation: ReconciliationProviderLookup, Compensation: CompensationContract{Mode: CompensationNone},
	}
	descriptor := ProviderDescriptor{
		ConnectorKey: "payments", ProviderKey: "acme", ProviderRevision: "provider-v1",
		Operations: []OperationDescriptor{
			reliabilityDescriptor("reserve", ModeEnqueue, reserve),
			reliabilityDescriptor("release", ModeEnqueue, release),
		},
	}
	if err := descriptor.Validate(); err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name string
		edit func(*ProviderDescriptor)
		want string
	}{
		{name: "missing", edit: func(value *ProviderDescriptor) { value.Operations = value.Operations[:1] }, want: "unknown compensation"},
		{name: "sync target", edit: func(value *ProviderDescriptor) { value.Operations[1].Mode = ModeCall }, want: "enqueue write"},
		{name: "reserve target", edit: func(value *ProviderDescriptor) { value.Operations[1].Reliability.Effect = EffectReserve }, want: "enqueue write"},
		{name: "non idempotent target", edit: func(value *ProviderDescriptor) {
			value.Operations[1].Reliability.Idempotency = IdempotencyContract{Strategy: IdempotencyNone}
		}, want: "idempotent and provider-reconcilable"},
		{name: "non reconcilable target", edit: func(value *ProviderDescriptor) { value.Operations[1].Reliability.Reconciliation = ReconciliationNone }, want: "idempotent and provider-reconcilable"},
		{name: "chain", edit: func(value *ProviderDescriptor) {
			value.Operations[1].Reliability.Compensation = CompensationContract{Mode: CompensationExplicit, OperationKey: "reserve"}
		}, want: "cannot declare another compensation"},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			value := descriptor
			value.Operations = append([]OperationDescriptor(nil), descriptor.Operations...)
			test.edit(&value)
			if err := value.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestReconcileResultHasClosedAuthoritativeOutcomes(t *testing.T) {
	valid := []ReconcileResult{
		{Outcome: ReconciliationSucceeded, Result: &CallResult{Payload: json.RawMessage(`{"status":"paid"}`), ResponseRef: "payment-1"}},
		{Outcome: ReconciliationFailed, FailureCode: "acme.payment_rejected"},
		{Outcome: ReconciliationPending, RetryAfter: time.Minute},
		{Outcome: ReconciliationNotFound},
		{Outcome: ReconciliationUnknown},
	}
	for _, result := range valid {
		if err := result.Validate(); err != nil {
			t.Fatalf("valid result %+v: %v", result, err)
		}
	}
	invalid := []ReconcileResult{
		{Outcome: "done"},
		{Outcome: ReconciliationSucceeded, FailureCode: "acme.failure"},
		{Outcome: ReconciliationFailed},
		{Outcome: ReconciliationFailed, FailureCode: "free text"},
		{Outcome: ReconciliationPending},
		{Outcome: ReconciliationNotFound, Result: &CallResult{ResponseRef: "receipt"}},
		{Outcome: ReconciliationUnknown, RetryAfter: time.Second},
	}
	for _, result := range invalid {
		if err := result.Validate(); err == nil {
			t.Fatalf("invalid result passed: %+v", result)
		}
	}
}

func TestReconcileRequestRequiresExactOriginalOperationIdentity(t *testing.T) {
	valid := ReconcileRequest{
		ConnectorKey: "payments", ProviderKey: "acme", OperationKey: "charge",
		ContractSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Connection:     Connection{ConnectorKey: "payments", ProviderKey: "acme"},
		RequestRef:     "charge-1", Payload: json.RawMessage(`{"amount":100}`), Timeout: time.Second,
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := []ReconcileRequest{
		{},
		func() ReconcileRequest { value := valid; value.ConnectorKey = " payments"; return value }(),
		func() ReconcileRequest { value := valid; value.OperationKey = " "; return value }(),
		func() ReconcileRequest { value := valid; value.ContractSHA256 = "bad"; return value }(),
		func() ReconcileRequest { value := valid; value.Connection.ProviderKey = "other"; return value }(),
		func() ReconcileRequest { value := valid; value.RequestRef = " request "; return value }(),
		func() ReconcileRequest { value := valid; value.Payload = json.RawMessage(`{`); return value }(),
		func() ReconcileRequest { value := valid; value.Timeout = -1; return value }(),
	}
	for _, request := range invalid {
		if err := request.Validate(); err == nil {
			t.Fatalf("invalid request passed: %+v", request)
		}
	}
}

type requiredReconcilerAdapter struct{ *mutableAdapter }

func (*requiredReconcilerAdapter) Reconcile(context.Context, ReconcileRequest) (ReconcileResult, error) {
	return ReconcileResult{Outcome: ReconciliationUnknown}, nil
}

func TestRegistryRequiresDeclaredProviderReconciliationCapability(t *testing.T) {
	descriptor := memberProvider(t).Descriptor()
	descriptor.Operations[0].Reliability.Effect = EffectWrite
	descriptor.Operations[0].Reliability.Idempotency = IdempotencyContract{Strategy: IdempotencyProviderKey, KeyRetentionSeconds: 3600}
	descriptor.Operations[0].Reliability.Reconciliation = ReconciliationProviderLookup
	plain := &mutableAdapter{descriptor: descriptor}
	if err := NewRegistry().Register(plain); err == nil || !strings.Contains(err.Error(), "does not implement Reconciler") {
		t.Fatalf("missing Reconciler error=%v", err)
	}
	registry := NewRegistry()
	provider := &requiredReconcilerAdapter{mutableAdapter: plain}
	if err := registry.Register(provider); err != nil {
		t.Fatal(err)
	}
	resolved, ok := registry.Provider("member_center", "acme")
	if !ok {
		t.Fatal("provider was not registered")
	}
	reconciler, ok := resolved.(Reconciler)
	if !ok {
		t.Fatalf("registered provider lost Reconciler: %T", resolved)
	}
	result, err := reconciler.Reconcile(t.Context(), ReconcileRequest{RequestRef: "request-1"})
	if err != nil || result.Outcome != ReconciliationUnknown {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}
