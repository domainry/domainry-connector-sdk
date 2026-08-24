package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

type memberRequest struct {
	MemberID string `json:"member_id"`
}

type memberResponse struct {
	Name string `json:"name"`
}

const getMemberContractSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const sendNoticeContractSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const syncMemberContractSHA256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

var readReliability = ReliabilityContract{
	Effect: EffectRead, Idempotency: IdempotencyContract{Strategy: IdempotencyNatural},
	Reconciliation: ReconciliationNone, Compensation: CompensationContract{Mode: CompensationNone},
}

var writeReliability = ReliabilityContract{
	Effect: EffectWrite, Idempotency: IdempotencyContract{Strategy: IdempotencyProviderKey, KeyRetentionSeconds: 86400},
	Reconciliation: ReconciliationNone, Compensation: CompensationContract{Mode: CompensationNone},
}

var getMember = CallOperation[memberRequest, memberResponse]{ConnectorKey: "member_center", ProviderKey: "acme", Key: "get_member", ContractSHA256: getMemberContractSHA256, Reliability: readReliability}

func memberProviderSchema() ProviderSchema {
	return ProviderSchema{ConnectorKey: "member_center", ProviderKey: "acme", ProviderRevision: "provider-v1"}
}

func memberProvider(t *testing.T) Adapter {
	t.Helper()
	bound, err := BindCall(getMember, func(_ context.Context, request TypedRequest[memberRequest]) (TypedResult[memberResponse], error) {
		return TypedResult[memberResponse]{Output: memberResponse{Name: "member:" + request.Input.MemberID}, ResponseRef: "member-ref"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(memberProviderSchema(), bound)
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestCallOperationKeepsTypedInputOutputAcrossErasedProviderBoundary(t *testing.T) {
	provider := memberProvider(t)
	payload, _ := json.Marshal(memberRequest{MemberID: "member-1"})
	result, err := provider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "get_member",
		ContractSHA256: getMemberContractSHA256, Mode: ModeCall, Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	var output memberResponse
	if err := json.Unmarshal(result.Payload, &output); err != nil || output.Name != "member:member-1" || result.ResponseRef != "member-ref" {
		t.Fatalf("output=%+v result=%+v error=%v", output, result, err)
	}
	badPayload := json.RawMessage(`{"member_id":"member-1","unknown":true}`)
	if _, err := provider.Call(t.Context(), CallRequest{ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "get_member", ContractSHA256: getMemberContractSHA256, Mode: ModeCall, Payload: badPayload}); err == nil {
		t.Fatal("unknown input fields must fail closed")
	}
}

type gatewayFunc func(context.Context, CallRequest) (CallResult, error)

func (f gatewayFunc) Call(ctx context.Context, request CallRequest) (CallResult, error) {
	return f(ctx, request)
}

type mutableAdapter struct {
	descriptor ProviderDescriptor
}

func (a *mutableAdapter) Descriptor() ProviderDescriptor { return a.descriptor }
func (a *mutableAdapter) Call(context.Context, CallRequest) (CallResult, error) {
	return CallResult{}, nil
}

type testableAdapter struct{ *mutableAdapter }

func (*testableAdapter) TestConnection(context.Context, TestConnectionRequest) (TestConnectionResult, error) {
	return TestConnectionResult{Connected: true}, nil
}

type webhookAdapter struct{ *mutableAdapter }

func (*webhookAdapter) VerifyWebhook(context.Context, VerifyWebhookRequest) (VerifiedWebhook, error) {
	return VerifiedWebhook{EventType: "member.updated", ExternalID: "event-1", Payload: json.RawMessage(`{"member_id":"member-1"}`)}, nil
}

type fullyCapableAdapter struct{ *mutableAdapter }

func (*fullyCapableAdapter) TestConnection(context.Context, TestConnectionRequest) (TestConnectionResult, error) {
	return TestConnectionResult{Connected: true}, nil
}

func (*fullyCapableAdapter) VerifyWebhook(context.Context, VerifyWebhookRequest) (VerifiedWebhook, error) {
	return VerifiedWebhook{EventType: "member.updated", ExternalID: "event-1", Payload: json.RawMessage(`{"member_id":"member-1"}`)}, nil
}

type allCapabilityAdapter struct{ *fullyCapableAdapter }

func (*allCapabilityAdapter) Reconcile(context.Context, ReconcileRequest) (ReconcileResult, error) {
	return ReconcileResult{Outcome: ReconciliationSucceeded}, nil
}

type reconcilerAdapter struct{ *mutableAdapter }

func (*reconcilerAdapter) Reconcile(context.Context, ReconcileRequest) (ReconcileResult, error) {
	return ReconcileResult{Outcome: ReconciliationSucceeded}, nil
}

type testableReconcilerAdapter struct{ *testableAdapter }

func (*testableReconcilerAdapter) Reconcile(context.Context, ReconcileRequest) (ReconcileResult, error) {
	return ReconcileResult{Outcome: ReconciliationSucceeded}, nil
}

type webhookReconcilerAdapter struct{ *webhookAdapter }

func (*webhookReconcilerAdapter) Reconcile(context.Context, ReconcileRequest) (ReconcileResult, error) {
	return ReconcileResult{Outcome: ReconciliationSucceeded}, nil
}

type validatingAdapter struct{ *mutableAdapter }

func (*validatingAdapter) ValidateConfig(Connection) error { return nil }

type validatingTestableAdapter struct{ *testableAdapter }

func (*validatingTestableAdapter) ValidateConfig(Connection) error { return nil }

type validatingWebhookAdapter struct{ *webhookAdapter }

func (*validatingWebhookAdapter) ValidateConfig(Connection) error { return nil }

type validatingFullyCapableAdapter struct{ *fullyCapableAdapter }

func (*validatingFullyCapableAdapter) ValidateConfig(Connection) error { return nil }

type validatingReconcilerAdapter struct{ *reconcilerAdapter }

func (*validatingReconcilerAdapter) ValidateConfig(Connection) error { return nil }

type validatingTestableReconcilerAdapter struct{ *testableReconcilerAdapter }

func (*validatingTestableReconcilerAdapter) ValidateConfig(Connection) error { return nil }

type validatingWebhookReconcilerAdapter struct{ *webhookReconcilerAdapter }

func (*validatingWebhookReconcilerAdapter) ValidateConfig(Connection) error { return nil }

type validatingAllCapabilityAdapter struct{ *allCapabilityAdapter }

func (*validatingAllCapabilityAdapter) ValidateConfig(Connection) error { return nil }

func TestCallInfersTypedOutput(t *testing.T) {
	gateway := gatewayFunc(func(_ context.Context, request CallRequest) (CallResult, error) {
		if request.OperationKey != "get_member" || request.ContractSHA256 != getMemberContractSHA256 {
			t.Fatalf("request=%+v", request)
		}
		return CallResult{Payload: json.RawMessage(`{"name":"Ada"}`)}, nil
	})
	output, err := Call(t.Context(), gateway, getMember, memberRequest{MemberID: "member-1"})
	if err != nil || output.Name != "Ada" {
		t.Fatalf("output=%+v error=%v", output, err)
	}
}

func TestOperationKindsBindExactModeAndResultShape(t *testing.T) {
	enqueue := EnqueueOperation[memberRequest]{
		ConnectorKey: "member_center", ProviderKey: "acme", Key: "send_notice", ContractSHA256: sendNoticeContractSHA256, Reliability: writeReliability,
	}
	start := StartOperation[memberRequest]{
		ConnectorKey: "member_center", ProviderKey: "acme", Key: "sync_member", ContractSHA256: syncMemberContractSHA256, Reliability: writeReliability,
	}
	boundEnqueue, err := BindEnqueueDelivery(enqueue, func(_ context.Context, request TypedRequest[memberRequest]) (DeliveryResult, error) {
		if request.Input.MemberID != "member-1" || !request.Delivery {
			t.Fatalf("enqueue request=%+v", request)
		}
		return DeliveryResult{ResponseRef: "delivery-1"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	boundStart, err := BindStartOperationDelivery(start, func(_ context.Context, request TypedRequest[memberRequest]) (DeliveryResult, error) {
		return DeliveryResult{ResponseRef: "remote-operation-1"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(memberProviderSchema(), boundEnqueue, boundStart)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(memberRequest{MemberID: "member-1"})
	delivery, err := provider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "send_notice",
		ContractSHA256: sendNoticeContractSHA256, Mode: ModeEnqueue, Payload: payload, Delivery: true,
	})
	if err != nil || len(delivery.Payload) != 0 || delivery.ResponseRef != "delivery-1" {
		t.Fatalf("delivery=%+v error=%v", delivery, err)
	}
	started, err := provider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "sync_member",
		ContractSHA256: syncMemberContractSHA256, Mode: ModeStartOperation, Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(started.Payload) != 0 || started.ResponseRef != "remote-operation-1" {
		t.Fatalf("start delivery=%+v", started)
	}
	if _, err := provider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "send_notice",
		ContractSHA256: sendNoticeContractSHA256, Mode: ModeCall, Payload: payload,
	}); err == nil {
		t.Fatal("operation mode mismatch must fail closed")
	}
	if _, err := provider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "send_notice",
		ContractSHA256: " " + sendNoticeContractSHA256, Mode: ModeEnqueue, Payload: payload,
	}); err == nil {
		t.Fatal("non-canonical invocation contract SHA-256 must fail closed")
	}
}

func TestBoundOperationsPreserveReceiptMetadataOnProviderError(t *testing.T) {
	quotaUsed := 100
	callHealth := &ResourceHealthReport{
		ObservationID: "quota-error-1", Kind: "quota", State: "exhausted", EvidenceSource: "provider_api",
		QuotaUsedPercent: &quotaUsed, CapabilityBlocked: true, ObservedAt: "2026-07-28T12:00:00Z",
	}
	call, err := BindCall(getMember, func(context.Context, TypedRequest[memberRequest]) (TypedResult[memberResponse], error) {
		return TypedResult[memberResponse]{
			Output:         memberResponse{Name: "must-not-be-published"},
			ResponseRef:    "provider-call-receipt",
			SecretUpdates:  map[string]string{"refresh_token": "rotated"},
			ResourceHealth: callHealth,
		}, UncertainError("acme.call_unknown", errors.New("provider response lost"))
	})
	if err != nil {
		t.Fatal(err)
	}
	callProvider, err := NewProvider(memberProviderSchema(), call)
	if err != nil {
		t.Fatal(err)
	}
	callResult, callErr := callProvider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "get_member",
		ContractSHA256: getMemberContractSHA256, Mode: ModeCall, Payload: json.RawMessage(`{"member_id":"member-1"}`),
	})
	if callErr == nil || callResult.ResponseRef != "provider-call-receipt" ||
		callResult.SecretUpdates["refresh_token"] != "rotated" || callResult.ResourceHealth != callHealth || len(callResult.Payload) != 0 {
		t.Fatalf("call result=%+v error=%v", callResult, callErr)
	}

	enqueue := EnqueueOperation[memberRequest]{
		ConnectorKey: "member_center", ProviderKey: "acme", Key: "send_notice",
		ContractSHA256: sendNoticeContractSHA256, Reliability: writeReliability,
	}
	delivery, err := BindEnqueueDelivery(enqueue, func(context.Context, TypedRequest[memberRequest]) (DeliveryResult, error) {
		return DeliveryResult{
			ResponseRef:    "provider-delivery-receipt",
			SecretUpdates:  map[string]string{"refresh_token": "rotated-again"},
			ResourceHealth: callHealth,
		}, UncertainError("acme.delivery_unknown", errors.New("provider acknowledgement lost"))
	})
	if err != nil {
		t.Fatal(err)
	}
	deliveryProvider, err := NewProvider(memberProviderSchema(), delivery)
	if err != nil {
		t.Fatal(err)
	}
	deliveryResult, deliveryErr := deliveryProvider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "send_notice",
		ContractSHA256: sendNoticeContractSHA256, Mode: ModeEnqueue, Delivery: true,
		Payload: json.RawMessage(`{"member_id":"member-1"}`),
	})
	if deliveryErr == nil || deliveryResult.ResponseRef != "provider-delivery-receipt" ||
		deliveryResult.SecretUpdates["refresh_token"] != "rotated-again" || deliveryResult.ResourceHealth != callHealth || len(deliveryResult.Payload) != 0 {
		t.Fatalf("delivery result=%+v error=%v", deliveryResult, deliveryErr)
	}
}

func TestBoundOperationsPreserveResourceHealthOnSuccessfulCallAndDelivery(t *testing.T) {
	quotaUsed := 82
	health := &ResourceHealthReport{
		ObservationID: "quota-success-1", Kind: "quota", State: "warning", EvidenceSource: "provider_api",
		QuotaUsedPercent: &quotaUsed, ObservedAt: "2026-07-28T12:00:00Z",
	}
	call, err := BindCall(getMember, func(context.Context, TypedRequest[memberRequest]) (TypedResult[memberResponse], error) {
		return TypedResult[memberResponse]{Output: memberResponse{Name: "Ada"}, ResourceHealth: health}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(memberProviderSchema(), call)
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "get_member",
		ContractSHA256: getMemberContractSHA256, Mode: ModeCall, Payload: json.RawMessage(`{"member_id":"member-1"}`),
	})
	if err != nil || result.ResourceHealth != health || len(result.Payload) == 0 {
		t.Fatalf("successful call result=%+v error=%v", result, err)
	}

	enqueue := EnqueueOperation[memberRequest]{ConnectorKey: "member_center", ProviderKey: "acme", Key: "send_notice", ContractSHA256: sendNoticeContractSHA256, Reliability: writeReliability}
	delivery, err := BindEnqueueDelivery(enqueue, func(context.Context, TypedRequest[memberRequest]) (DeliveryResult, error) {
		return DeliveryResult{ResponseRef: "delivery", ResourceHealth: health}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err = NewProvider(memberProviderSchema(), delivery)
	if err != nil {
		t.Fatal(err)
	}
	result, err = provider.Call(t.Context(), CallRequest{
		ConnectorKey: "member_center", ProviderKey: "acme", OperationKey: "send_notice",
		ContractSHA256: sendNoticeContractSHA256, Mode: ModeEnqueue, Delivery: true, Payload: json.RawMessage(`{"member_id":"member-1"}`),
	})
	if err != nil || result.ResourceHealth != health || len(result.Payload) != 0 {
		t.Fatalf("successful delivery result=%+v error=%v", result, err)
	}
}

func TestOperationContractSHA256MustBeCanonical(t *testing.T) {
	operation := CallOperation[memberRequest, memberResponse]{
		ConnectorKey: "member_center", ProviderKey: "acme", Key: "get_member", ContractSHA256: " " + getMemberContractSHA256, Reliability: readReliability,
	}
	if _, err := BindCall(operation, func(context.Context, TypedRequest[memberRequest]) (TypedResult[memberResponse], error) {
		return TypedResult[memberResponse]{}, nil
	}); err == nil {
		t.Fatal("non-canonical contract SHA-256 must be rejected")
	}
}

func TestProviderConfigAndSecretSchemaIsClosedAndImmutable(t *testing.T) {
	minimum, maximum := 1.0, 120.0
	schema := ProviderSchema{
		ConnectorKey: "member_center", ProviderKey: "acme", ProviderRevision: "provider-v2",
		ConfigFields: []ConfigField{
			{Key: "region", Name: "Region", Type: ConfigFieldSelect, Required: true, Default: json.RawMessage(`"us"`), Validation: ConfigValidation{Options: []string{"us", "eu"}}, I18n: map[string]FieldLocalization{"zh-CN": {Name: "区域"}}},
			{Key: "timeout_seconds", Name: "Timeout", Type: ConfigFieldInteger, Default: json.RawMessage(`30`), Validation: ConfigValidation{Min: &minimum, Max: &maximum}, RequiredWith: []string{"region"}},
		},
		SecretFields: []SecretField{{Key: "api_key", Name: "API key", Required: true, CredentialKind: SecretCredentialAPIKey, MaterialFormat: SecretMaterialOpaque, RotationPolicy: SecretRotationManual, ExpiryPolicy: SecretExpiryOptional, TestRequirement: SecretTestWhenBound}},
	}
	bound, err := BindCall(getMember, func(context.Context, TypedRequest[memberRequest]) (TypedResult[memberResponse], error) {
		return TypedResult[memberResponse]{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(schema, bound)
	if err != nil {
		t.Fatal(err)
	}
	schema.ConfigFields[0].Validation.Options[0] = "mutated"
	schema.ConfigFields[0].I18n["zh-CN"] = FieldLocalization{Name: "mutated"}
	*schema.ConfigFields[1].Validation.Min = 99
	schema.SecretFields[0].CredentialKind = "mutated"
	first := provider.Descriptor()
	if first.ConfigFields[0].Validation.Options[0] != "us" || first.ConfigFields[0].I18n["zh-CN"].Name != "区域" || *first.ConfigFields[1].Validation.Min != 1 || first.SecretFields[0].CredentialKind != SecretCredentialAPIKey {
		t.Fatalf("provider descriptor was mutated through input schema: %+v", first)
	}
	first.ConfigFields[0].Validation.Options[0] = "mutated-again"
	*first.ConfigFields[1].Validation.Min = 88
	first.SecretFields[0].CredentialKind = "mutated-again"
	second := provider.Descriptor()
	if second.ConfigFields[0].Validation.Options[0] != "us" || *second.ConfigFields[1].Validation.Min != 1 || second.SecretFields[0].CredentialKind != SecretCredentialAPIKey {
		t.Fatalf("provider descriptor was mutated through returned snapshot: %+v", second)
	}
}

func TestApplyConfigDefaultsPreservesExplicitValuesAndClonesJSON(t *testing.T) {
	fields := []ConfigField{
		{Key: "base_url", Type: ConfigFieldText, Default: json.RawMessage(`"https://provider.example"`)},
		{Key: "timeout_seconds", Type: ConfigFieldInteger, Default: json.RawMessage(`15`)},
		{Key: "options", Type: ConfigFieldJSON, Default: json.RawMessage(`{"region":"JP"}`)},
		{Key: "conditional", Type: ConfigFieldText, Default: json.RawMessage(`"ready"`), RequiredWith: []string{"dependency"}},
	}
	first, err := ApplyConfigDefaults(fields, map[string]any{"base_url": "https://tenant.example", "dependency": "set"})
	if err != nil {
		t.Fatal(err)
	}
	if first["base_url"] != "https://tenant.example" || first["timeout_seconds"] != json.Number("15") || first["conditional"] != "ready" {
		t.Fatalf("materialized defaults=%#v", first)
	}
	first["options"].(map[string]any)["region"] = "mutated"
	second, err := ApplyConfigDefaults(fields, nil)
	if err != nil {
		t.Fatal(err)
	}
	if second["options"].(map[string]any)["region"] != "JP" {
		t.Fatalf("mutable default leaked across materializations: %#v", second)
	}
	if _, exists := second["conditional"]; exists {
		t.Fatalf("required_with default materialized without dependency: %#v", second)
	}
	explicitEmpty, err := ApplyConfigDefaults(fields, map[string]any{"base_url": ""})
	if err != nil || explicitEmpty["base_url"] != "" {
		t.Fatalf("explicit empty value was overwritten: %#v error=%v", explicitEmpty, err)
	}
	if _, err := ApplyConfigDefaults([]ConfigField{{Key: "bad", Default: json.RawMessage(`1 2`)}}, nil); err == nil {
		t.Fatal("invalid default was not rejected fail-closed")
	}
}

func TestProviderSchemaRejectsOpenOrInconsistentFieldContracts(t *testing.T) {
	validSecret := SecretField{Key: "api_key", Name: "API key", CredentialKind: SecretCredentialAPIKey, MaterialFormat: SecretMaterialOpaque, RotationPolicy: SecretRotationManual, ExpiryPolicy: SecretExpiryOptional, TestRequirement: SecretTestOptional}
	minimum := 2.0
	tests := map[string]ProviderSchema{
		"invalid config type":            {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "value", Name: "Value", Type: "dynamic"}}},
		"invalid default type":           {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "count", Name: "Count", Type: ConfigFieldInteger, Default: json.RawMessage(`"one"`)}}},
		"trailing default value":         {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "count", Name: "Count", Type: ConfigFieldInteger, Default: json.RawMessage(`1 2`)}}},
		"select without options":         {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "region", Name: "Region", Type: ConfigFieldSelect}}},
		"select default outside options": {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "region", Name: "Region", Type: ConfigFieldSelect, Default: json.RawMessage(`"apac"`), Validation: ConfigValidation{Options: []string{"us", "eu"}}}}},
		"default below minimum":          {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "count", Name: "Count", Type: ConfigFieldInteger, Default: json.RawMessage(`1`), Validation: ConfigValidation{Min: &minimum}}}},
		"invalid pattern":                {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "region", Name: "Region", Type: ConfigFieldText, Validation: ConfigValidation{Pattern: "["}}}},
		"unknown dependency":             {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "region", Name: "Region", Type: ConfigFieldText, RequiredWith: []string{"missing"}}}},
		"invalid secret metadata":        {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", SecretFields: []SecretField{{Key: "api_key", Name: "API key"}}},
		"config secret collision":        {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", ConfigFields: []ConfigField{{Key: "api_key", Name: "API key", Type: ConfigFieldText}}, SecretFields: []SecretField{validSecret}},
		"invalid startup activation":     {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", StartupActivation: "automatic"},
		"default safe missing default":   {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", StartupActivation: StartupActivationDefaultSafe, ConfigFields: []ConfigField{{Key: "endpoint", Name: "Endpoint", Type: ConfigFieldText, Required: true}}},
		"default safe required secret":   {ConnectorKey: "member", ProviderKey: "acme", ProviderRevision: "v1", StartupActivation: StartupActivationDefaultSafe, SecretFields: []SecretField{{Key: "api_key", Name: "API key", Required: true, CredentialKind: SecretCredentialAPIKey, MaterialFormat: SecretMaterialOpaque, RotationPolicy: SecretRotationManual, ExpiryPolicy: SecretExpiryOptional, TestRequirement: SecretTestOptional}}},
	}
	for name, schema := range tests {
		t.Run(name, func(t *testing.T) {
			if err := schema.Validate(); err == nil {
				t.Fatalf("invalid schema was accepted: %+v", schema)
			}
		})
	}
}

func TestProviderRegistryUsesExactPairAndExtensionRegistrationIsAtomic(t *testing.T) {
	provider := memberProvider(t)
	registry := NewRegistry()
	if err := registry.RegisterProviderSet(ProviderSet{Providers: []Adapter{provider}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Provider("member_center", "acme"); !ok {
		t.Fatal("exact connector/provider pair must resolve")
	}
	if _, ok := registry.Provider("member_center", ""); ok {
		t.Fatal("provider fallback must not resolve")
	}
	if err := registry.RegisterProviderSet(ProviderSet{Providers: []Adapter{provider, provider}}); !errors.Is(err, ErrProviderDuplicate) {
		t.Fatalf("duplicate error=%v", err)
	}
	if len(registry.Descriptors()) != 1 {
		t.Fatalf("registry changed after atomic failure: %+v", registry.Descriptors())
	}
	registry.Freeze()
	if err := registry.Register(provider); !errors.Is(err, ErrRegistryFrozen) {
		t.Fatalf("frozen error=%v", err)
	}
}

func TestProviderRegistryFreezesCustomAdapterDescriptor(t *testing.T) {
	provider := memberProvider(t)
	custom := &mutableAdapter{descriptor: provider.Descriptor()}
	custom.descriptor.ConfigFields = []ConfigField{{Key: "region", Name: "Region", Type: ConfigFieldText}}
	registry := NewRegistry()
	if err := registry.Register(custom); err != nil {
		t.Fatal(err)
	}
	custom.descriptor.ConfigFields[0].Name = "mutated"
	custom.descriptor.Operations[0].Key = "mutated"
	descriptor := registry.Descriptors()[0]
	if descriptor.ConfigFields[0].Name != "Region" || descriptor.Operations[0].Key != "get_member" {
		t.Fatalf("registered descriptor remained mutable: %+v", descriptor)
	}
	resolved, ok := registry.Provider("member_center", "acme")
	if !ok || resolved == custom {
		t.Fatalf("registry did not retain a frozen adapter wrapper: %T %v", resolved, ok)
	}
}

func TestProviderRegistryPreservesOnlyCapabilitiesActuallyImplemented(t *testing.T) {
	descriptor := memberProvider(t).Descriptor()
	tests := []struct {
		name           string
		adapter        Adapter
		wantValidator  bool
		wantTester     bool
		wantWebhook    bool
		wantReconciler bool
	}{
		{name: "call only", adapter: &mutableAdapter{descriptor: descriptor}},
		{name: "connection tester", adapter: &testableAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}, wantTester: true},
		{name: "webhook verifier", adapter: &webhookAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}, wantWebhook: true},
		{name: "tester and webhook", adapter: &fullyCapableAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}, wantTester: true, wantWebhook: true},
		{name: "reconciler", adapter: &reconcilerAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}, wantReconciler: true},
		{name: "tester and reconciler", adapter: &testableReconcilerAdapter{testableAdapter: &testableAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}, wantTester: true, wantReconciler: true},
		{name: "webhook and reconciler", adapter: &webhookReconcilerAdapter{webhookAdapter: &webhookAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}, wantWebhook: true, wantReconciler: true},
		{name: "all", adapter: &allCapabilityAdapter{fullyCapableAdapter: &fullyCapableAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}, wantTester: true, wantWebhook: true, wantReconciler: true},
		{name: "validator", adapter: &validatingAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}, wantValidator: true},
		{name: "validator and tester", adapter: &validatingTestableAdapter{testableAdapter: &testableAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}, wantValidator: true, wantTester: true},
		{name: "validator and webhook", adapter: &validatingWebhookAdapter{webhookAdapter: &webhookAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}, wantValidator: true, wantWebhook: true},
		{name: "validator tester and webhook", adapter: &validatingFullyCapableAdapter{fullyCapableAdapter: &fullyCapableAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}, wantValidator: true, wantTester: true, wantWebhook: true},
		{name: "validator and reconciler", adapter: &validatingReconcilerAdapter{reconcilerAdapter: &reconcilerAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}, wantValidator: true, wantReconciler: true},
		{name: "validator tester and reconciler", adapter: &validatingTestableReconcilerAdapter{testableReconcilerAdapter: &testableReconcilerAdapter{testableAdapter: &testableAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}}, wantValidator: true, wantTester: true, wantReconciler: true},
		{name: "validator webhook and reconciler", adapter: &validatingWebhookReconcilerAdapter{webhookReconcilerAdapter: &webhookReconcilerAdapter{webhookAdapter: &webhookAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}}, wantValidator: true, wantWebhook: true, wantReconciler: true},
		{name: "all with validator", adapter: &validatingAllCapabilityAdapter{allCapabilityAdapter: &allCapabilityAdapter{fullyCapableAdapter: &fullyCapableAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}}}, wantValidator: true, wantTester: true, wantWebhook: true, wantReconciler: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := NewRegistry()
			if err := registry.Register(test.adapter); err != nil {
				t.Fatal(err)
			}
			resolved, ok := registry.Provider("member_center", "acme")
			if !ok {
				t.Fatal("provider was not registered")
			}
			_, hasValidator := resolved.(ConfigValidator)
			_, hasTester := resolved.(ConnectionTester)
			_, hasWebhook := resolved.(WebhookVerifier)
			_, hasReconciler := resolved.(Reconciler)
			if hasValidator != test.wantValidator || hasTester != test.wantTester || hasWebhook != test.wantWebhook || hasReconciler != test.wantReconciler {
				t.Fatalf("capabilities validator=%v tester=%v webhook=%v reconciler=%v", hasValidator, hasTester, hasWebhook, hasReconciler)
			}
			if hasValidator {
				if err := resolved.(ConfigValidator).ValidateConfig(Connection{}); err != nil {
					t.Fatalf("registered config validator error=%v", err)
				}
			}
			if hasTester {
				result, err := resolved.(ConnectionTester).TestConnection(t.Context(), TestConnectionRequest{})
				if err != nil || !result.Connected {
					t.Fatalf("registered connection tester result=%+v error=%v", result, err)
				}
			}
			if hasWebhook {
				result, err := resolved.(WebhookVerifier).VerifyWebhook(t.Context(), VerifyWebhookRequest{})
				if err != nil || result.ExternalID != "event-1" {
					t.Fatalf("registered webhook verifier result=%+v error=%v", result, err)
				}
			}
			if hasReconciler {
				result, err := resolved.(Reconciler).Reconcile(t.Context(), ReconcileRequest{})
				if err != nil || result.Outcome != ReconciliationSucceeded {
					t.Fatalf("registered reconciler result=%+v error=%v", result, err)
				}
			}
		})
	}
}

func TestContractIdentityIsStable(t *testing.T) {
	if ContractVersion != "connector-contract-v10" {
		t.Fatalf("connector contract version=%q", ContractVersion)
	}
	if actual := ComputedContractSHA256(); actual != ContractSHA256 {
		t.Fatalf("connector contract hash drifted: got %s, update only with an intentional contract version change", actual)
	}
}

func TestOptionalCapabilitiesHaveIndependentClosedEnvelopes(t *testing.T) {
	providers := reflect.TypeOf(ProviderSet{})
	if providers.NumField() != 1 || providers.Field(0).Name != "Providers" {
		t.Fatalf("ProviderSet must expose only Provider adapters, fields=%v", providers.NumField())
	}
	validator := reflect.TypeOf((*ConfigValidator)(nil)).Elem()
	if validator.NumMethod() != 1 {
		t.Fatalf("ConfigValidator methods=%d", validator.NumMethod())
	}
	if _, exists := validator.MethodByName("ValidateConfig"); !exists {
		t.Fatal("ConfigValidator is missing ValidateConfig")
	}
	tester := reflect.TypeOf((*ConnectionTester)(nil)).Elem()
	if tester.NumMethod() != 1 {
		t.Fatalf("ConnectionTester methods=%d", tester.NumMethod())
	}
	if _, exists := tester.MethodByName("TestConnection"); !exists {
		t.Fatal("ConnectionTester is missing TestConnection")
	}
	verifier := reflect.TypeOf((*WebhookVerifier)(nil)).Elem()
	if verifier.NumMethod() != 1 {
		t.Fatalf("WebhookVerifier methods=%d", verifier.NumMethod())
	}
	if _, exists := verifier.MethodByName("VerifyWebhook"); !exists {
		t.Fatal("WebhookVerifier is missing VerifyWebhook")
	}
	reconciler := reflect.TypeOf((*Reconciler)(nil)).Elem()
	if reconciler.NumMethod() != 1 {
		t.Fatalf("Reconciler methods=%d", reconciler.NumMethod())
	}
	if _, exists := reconciler.MethodByName("Reconcile"); !exists {
		t.Fatal("Reconciler is missing Reconcile")
	}
	types := []struct {
		value  any
		fields []string
	}{
		{value: TestConnectionRequest{}, fields: []string{"ConnectorKey", "ProviderKey", "Connection", "Secrets", "Timeout", "Principal"}},
		{value: TestConnectionResult{}, fields: []string{"Connected", "Details", "SecretUpdates"}},
		{value: VerifyWebhookRequest{}, fields: []string{"ConnectorKey", "ProviderKey", "Connection", "Headers", "Query", "Secrets", "Body", "ReceivedAt"}},
		{value: VerifiedWebhook{}, fields: []string{"EventType", "ExternalID", "Payload", "Security", "Challenge", "ChallengeFormat", "ExternalIdentity", "DeliveryReceipt"}},
		{value: ReconcileRequest{}, fields: []string{"ConnectorKey", "ProviderKey", "OperationKey", "ContractSHA256", "Connection", "RequestRef", "ResponseRef", "Payload", "Secrets", "Timeout", "Principal"}},
		{value: ReconcileResult{}, fields: []string{"Outcome", "Result", "FailureCode", "RetryAfter"}},
	}
	for _, definition := range types {
		current := reflect.TypeOf(definition.value)
		for _, field := range definition.fields {
			if _, exists := current.FieldByName(field); !exists {
				t.Fatalf("%s is missing %s", current.Name(), field)
			}
		}
	}
}

func TestTransportContractHasOnlyApprovedOutboundOperations(t *testing.T) {
	transport := reflect.TypeOf((*Transport)(nil)).Elem()
	if transport.NumMethod() != 2 {
		t.Fatalf("Transport methods=%d", transport.NumMethod())
	}
	for _, name := range []string{"ExecuteSQL", "RoundTripHTTP"} {
		if _, ok := transport.MethodByName(name); !ok {
			t.Fatalf("Transport is missing %s", name)
		}
	}
}

func TestHTTPRequestSecretsAreNotSerialized(t *testing.T) {
	payload, err := json.Marshal(HTTPRequest{
		URL:           "https://example.test/resource",
		SecretHeaders: map[string][]string{"Authorization": {"Bearer resolved-secret"}},
		SecretQuery:   map[string]string{"apiKey": "resolved-secret"},
		SecretForm:    map[string]string{"refresh_token": "resolved-secret"},
		SecretJSON:    map[string]string{"api_key": "resolved-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte("apiKey")) || bytes.Contains(payload, []byte("Authorization")) || bytes.Contains(payload, []byte("refresh_token")) || bytes.Contains(payload, []byte("resolved-secret")) {
		t.Fatalf("serialized HTTPRequest leaked secret material: %s", payload)
	}
	for _, field := range []string{"SecretHeaders", "SecretQuery", "SecretForm", "SecretJSON"} {
		if _, exists := reflect.TypeOf(HTTPRequest{}).FieldByName(field); !exists {
			t.Fatalf("HTTPRequest is missing %s", field)
		}
	}
}

func TestSMTPPasswordIsNotSerialized(t *testing.T) {
	payload, err := json.Marshal(SMTPRequest{Host: "smtp.example.test", Username: "user", SecretPassword: "resolved-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte("resolved-secret")) || bytes.Contains(payload, []byte("SecretPassword")) {
		t.Fatalf("serialized SMTPRequest leaked secret material: %s", payload)
	}
	transport := reflect.TypeOf((*SMTPTransport)(nil)).Elem()
	if transport.NumMethod() != 1 {
		t.Fatalf("SMTPTransport methods=%d", transport.NumMethod())
	}
}

func TestMQTTSecretsAreNotSerialized(t *testing.T) {
	payload, err := json.Marshal(MQTTRequest{BrokerURL: "mqtts://broker.example", Topic: "devices/1", SecretUsername: "user-secret", SecretPassword: "password-secret", SecretCertificate: "certificate-secret", SecretPrivateKey: "private-key-secret"})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"user-secret", "password-secret", "certificate-secret", "private-key-secret"} {
		if bytes.Contains(payload, []byte(secret)) {
			t.Fatalf("serialized MQTTRequest leaked secret material: %s", payload)
		}
	}
	transport := reflect.TypeOf((*MQTTTransport)(nil)).Elem()
	if transport.NumMethod() != 1 || transport.Method(0).Name != "ExecuteMQTT" {
		t.Fatalf("MQTTTransport=%v", transport)
	}
}

func TestFilesystemTransportIsAnExplicitOptionalCapability(t *testing.T) {
	transport := reflect.TypeOf((*FilesystemTransport)(nil)).Elem()
	if transport.NumMethod() != 1 || transport.Method(0).Name != "ExecuteFilesystem" {
		t.Fatalf("FilesystemTransport=%v", transport)
	}
	for _, operation := range []FilesystemOperation{FilesystemOperationProbe, FilesystemOperationRead, FilesystemOperationWrite, FilesystemOperationDelete} {
		if operation == "" {
			t.Fatal("filesystem operation must be stable")
		}
	}
}

func TestProcessTransportExposesOnlyGovernedLineSessions(t *testing.T) {
	transport := reflect.TypeOf((*ProcessTransport)(nil)).Elem()
	if transport.NumMethod() != 1 || transport.Method(0).Name != "StartProcess" {
		t.Fatalf("ProcessTransport=%v", transport)
	}
	session := reflect.TypeOf((*ProcessSession)(nil)).Elem()
	if session.NumMethod() != 3 {
		t.Fatalf("ProcessSession methods=%d", session.NumMethod())
	}
	for _, method := range []string{"Close", "ReceiveLine", "SendLine"} {
		if _, ok := session.MethodByName(method); !ok {
			t.Fatalf("ProcessSession is missing %s", method)
		}
	}
}

func TestAdapterAndCallEnvelopeExposeTheStablePublicBoundary(t *testing.T) {
	adapter := reflect.TypeOf((*Adapter)(nil)).Elem()
	if adapter.Name() != "Adapter" || adapter.NumMethod() != 2 {
		t.Fatalf("Adapter type=%v methods=%d", adapter, adapter.NumMethod())
	}
	for _, methodName := range []string{"Call", "Descriptor"} {
		if _, exists := adapter.MethodByName(methodName); !exists {
			t.Fatalf("Adapter is missing %s", methodName)
		}
	}
	request := reflect.TypeOf(CallRequest{})
	for _, fieldName := range []string{"ConnectorKey", "ProviderKey", "OperationKey", "ContractSHA256", "Mode", "Connection", "Payload", "RequestRef", "Headers", "Secrets", "Delivery", "Timeout", "Principal"} {
		if _, exists := request.FieldByName(fieldName); !exists {
			t.Fatalf("CallRequest is missing %s", fieldName)
		}
	}
	result := reflect.TypeOf(CallResult{})
	for _, fieldName := range []string{"Payload", "ResponseRef", "SecretUpdates", "ResourceHealth"} {
		if _, exists := result.FieldByName(fieldName); !exists {
			t.Fatalf("CallResult is missing %s", fieldName)
		}
	}
	descriptor := reflect.TypeOf(ProviderDescriptor{})
	for _, fieldName := range []string{"ConnectorKey", "ProviderKey", "ProviderRevision", "ConfigFields", "SecretFields", "Operations"} {
		if _, exists := descriptor.FieldByName(fieldName); !exists {
			t.Fatalf("ProviderDescriptor is missing %s", fieldName)
		}
	}
	operation := reflect.TypeOf(OperationDescriptor{})
	if _, exists := operation.FieldByName("Reliability"); !exists {
		t.Fatal("OperationDescriptor is missing Reliability")
	}
}
