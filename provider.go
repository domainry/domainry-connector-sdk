package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ProviderDescriptor struct {
	ConnectorKey      string                  `json:"connector_key"`
	ProviderKey       string                  `json:"provider_key"`
	ProviderRevision  string                  `json:"provider_revision"`
	StartupActivation StartupActivationPolicy `json:"startup_activation,omitempty"`
	ConfigFields      []ConfigField           `json:"config_fields,omitempty"`
	SecretFields      []SecretField           `json:"secret_fields,omitempty"`
	Operations        []OperationDescriptor   `json:"operations"`
}

func (d ProviderDescriptor) Validate() error {
	if err := (ProviderSchema{ConnectorKey: d.ConnectorKey, ProviderKey: d.ProviderKey, ProviderRevision: d.ProviderRevision, StartupActivation: d.StartupActivation, ConfigFields: d.ConfigFields, SecretFields: d.SecretFields}).Validate(); err != nil {
		return err
	}
	if len(d.Operations) == 0 {
		return fmt.Errorf("connector provider %s/%s requires at least one operation", d.ConnectorKey, d.ProviderKey)
	}
	seen := map[string]OperationDescriptor{}
	for _, operation := range d.Operations {
		if err := operation.Validate(); err != nil {
			return err
		}
		if _, exists := seen[operation.Key]; exists {
			return fmt.Errorf("connector provider %s/%s operation %s is duplicated", d.ConnectorKey, d.ProviderKey, operation.Key)
		}
		seen[operation.Key] = operation
	}
	for _, operation := range d.Operations {
		if operation.Reliability.Compensation.Mode == CompensationNone {
			continue
		}
		compensationKey := strings.TrimSpace(operation.Reliability.Compensation.OperationKey)
		compensation, exists := seen[compensationKey]
		if !exists {
			return fmt.Errorf("connector provider %s/%s operation %s references unknown compensation operation %s", d.ConnectorKey, d.ProviderKey, operation.Key, compensationKey)
		}
		if compensation.Mode != ModeEnqueue || compensation.Reliability.Effect != EffectWrite {
			return fmt.Errorf("connector provider %s/%s compensation operation %s must be an enqueue write", d.ConnectorKey, d.ProviderKey, compensationKey)
		}
		if compensation.Reliability.Idempotency.Strategy == IdempotencyNone || compensation.Reliability.Reconciliation != ReconciliationProviderLookup {
			return fmt.Errorf("connector provider %s/%s compensation operation %s must be idempotent and provider-reconcilable", d.ConnectorKey, d.ProviderKey, compensationKey)
		}
		if compensation.Reliability.Compensation.Mode != CompensationNone {
			return fmt.Errorf("connector provider %s/%s compensation operation %s cannot declare another compensation", d.ConnectorKey, d.ProviderKey, compensationKey)
		}
	}
	return nil
}

func (d ProviderDescriptor) RequiresReconciler() bool {
	for _, operation := range d.Operations {
		if operation.Reliability.Reconciliation == ReconciliationProviderLookup {
			return true
		}
	}
	return false
}

type boundProvider struct {
	descriptor ProviderDescriptor
	operations map[string]BoundOperation
}

func NewProvider(schema ProviderSchema, operations ...BoundOperation) (Adapter, error) {
	if err := schema.Validate(); err != nil {
		return nil, err
	}
	descriptor := ProviderDescriptor{
		ConnectorKey: schema.ConnectorKey, ProviderKey: schema.ProviderKey, ProviderRevision: schema.ProviderRevision, StartupActivation: schema.StartupActivation,
		ConfigFields: cloneConfigFields(schema.ConfigFields), SecretFields: cloneSecretFields(schema.SecretFields),
		Operations: make([]OperationDescriptor, 0, len(operations)),
	}
	byKey := make(map[string]BoundOperation, len(operations))
	for _, operation := range operations {
		current := operation.Descriptor()
		if current.ConnectorKey != descriptor.ConnectorKey || current.ProviderKey != descriptor.ProviderKey {
			return nil, fmt.Errorf("connector operation %s belongs to %s/%s, not %s/%s", current.Key, current.ConnectorKey, current.ProviderKey, descriptor.ConnectorKey, descriptor.ProviderKey)
		}
		descriptor.Operations = append(descriptor.Operations, current)
		byKey[current.Key] = operation
	}
	sort.Slice(descriptor.Operations, func(i, j int) bool { return descriptor.Operations[i].Key < descriptor.Operations[j].Key })
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	return &boundProvider{descriptor: descriptor, operations: byKey}, nil
}

func (p *boundProvider) Descriptor() ProviderDescriptor { return cloneProviderDescriptor(p.descriptor) }

func (p *boundProvider) Call(ctx context.Context, request CallRequest) (CallResult, error) {
	if strings.TrimSpace(request.ConnectorKey) != p.descriptor.ConnectorKey || strings.TrimSpace(request.ProviderKey) != p.descriptor.ProviderKey {
		return CallResult{}, fmt.Errorf("connector provider mismatch: got %s/%s, want %s/%s", request.ConnectorKey, request.ProviderKey, p.descriptor.ConnectorKey, p.descriptor.ProviderKey)
	}
	operation, ok := p.operations[strings.TrimSpace(request.OperationKey)]
	if !ok {
		return CallResult{}, fmt.Errorf("connector provider %s/%s operation %s is not registered", p.descriptor.ConnectorKey, p.descriptor.ProviderKey, request.OperationKey)
	}
	return operation.invoke(ctx, request)
}

func cloneProviderDescriptor(descriptor ProviderDescriptor) ProviderDescriptor {
	descriptor.ConfigFields = cloneConfigFields(descriptor.ConfigFields)
	descriptor.SecretFields = cloneSecretFields(descriptor.SecretFields)
	descriptor.Operations = append([]OperationDescriptor(nil), descriptor.Operations...)
	return descriptor
}

func cloneConfigFields(fields []ConfigField) []ConfigField {
	result := make([]ConfigField, len(fields))
	for index, field := range fields {
		field.Default = append(json.RawMessage(nil), field.Default...)
		field.Validation.Options = append([]string(nil), field.Validation.Options...)
		field.Validation.Min = cloneFloat64(field.Validation.Min)
		field.Validation.Max = cloneFloat64(field.Validation.Max)
		field.RequiredWith = append([]string(nil), field.RequiredWith...)
		field.I18n = cloneFieldLocalizations(field.I18n)
		result[index] = field
	}
	return result
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneSecretFields(fields []SecretField) []SecretField {
	result := make([]SecretField, len(fields))
	for index, field := range fields {
		field.I18n = cloneFieldLocalizations(field.I18n)
		result[index] = field
	}
	return result
}

func cloneFieldLocalizations(values map[string]FieldLocalization) map[string]FieldLocalization {
	if values == nil {
		return nil
	}
	result := make(map[string]FieldLocalization, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
