package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var operationContractSHA256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type OperationDescriptor struct {
	ConnectorKey   string              `json:"connector_key"`
	ProviderKey    string              `json:"provider_key"`
	Key            string              `json:"key"`
	Mode           OperationMode       `json:"mode"`
	ContractSHA256 string              `json:"contract_sha256"`
	Reliability    ReliabilityContract `json:"reliability"`
}

func (d OperationDescriptor) Validate() error {
	if strings.TrimSpace(d.ConnectorKey) == "" || strings.TrimSpace(d.ProviderKey) == "" {
		return errors.New("connector operation connector and provider keys are required")
	}
	if strings.TrimSpace(d.Key) == "" {
		return errors.New("connector operation key is required")
	}
	switch d.Mode {
	case ModeCall, ModeEnqueue, ModeStartOperation:
	default:
		return fmt.Errorf("connector operation %s has unsupported mode %q", d.Key, d.Mode)
	}
	if !operationContractSHA256Pattern.MatchString(d.ContractSHA256) {
		return fmt.Errorf("connector operation %s contract SHA-256 is invalid", d.Key)
	}
	if err := d.Reliability.Validate(d.Key); err != nil {
		return err
	}
	return nil
}

// CallOperation is a synchronous operation whose typed output is required by
// the caller before it can continue.
type CallOperation[I any, O any] struct {
	ConnectorKey   string
	ProviderKey    string
	Key            string
	ContractSHA256 string
	Reliability    ReliabilityContract
}

func (o CallOperation[I, O]) Descriptor() OperationDescriptor {
	return OperationDescriptor{
		ConnectorKey: strings.TrimSpace(o.ConnectorKey), ProviderKey: strings.TrimSpace(o.ProviderKey),
		Key: strings.TrimSpace(o.Key), Mode: ModeCall, ContractSHA256: o.ContractSHA256, Reliability: o.Reliability,
	}
}

// EnqueueOperation is an asynchronous command. Its provider result is delivery
// metadata only; no typed output is returned to the Action Handler.
type EnqueueOperation[I any] struct {
	ConnectorKey   string
	ProviderKey    string
	Key            string
	ContractSHA256 string
	Reliability    ReliabilityContract
}

func (o EnqueueOperation[I]) Descriptor() OperationDescriptor {
	return OperationDescriptor{
		ConnectorKey: strings.TrimSpace(o.ConnectorKey), ProviderKey: strings.TrimSpace(o.ProviderKey),
		Key: strings.TrimSpace(o.Key), Mode: ModeEnqueue, ContractSHA256: o.ContractSHA256, Reliability: o.Reliability,
	}
}

// StartOperation describes asynchronous delivery that starts a long-running
// external operation. Runtime owns the local durable operation identity.
type StartOperation[I any] struct {
	ConnectorKey   string
	ProviderKey    string
	Key            string
	ContractSHA256 string
	Reliability    ReliabilityContract
}

func (o StartOperation[I]) Descriptor() OperationDescriptor {
	return OperationDescriptor{
		ConnectorKey: strings.TrimSpace(o.ConnectorKey), ProviderKey: strings.TrimSpace(o.ProviderKey),
		Key: strings.TrimSpace(o.Key), Mode: ModeStartOperation, ContractSHA256: o.ContractSHA256, Reliability: o.Reliability,
	}
}

type TypedRequest[I any] struct {
	Input      I
	Connection Connection
	RequestRef string
	Headers    map[string]string
	Secrets    map[string]string
	Delivery   bool
	Principal  Principal
}

type TypedResult[O any] struct {
	Output         O
	ResponseRef    string
	SecretUpdates  map[string]string
	ResourceHealth *ResourceHealthReport
}

type CallHandler[I any, O any] func(context.Context, TypedRequest[I]) (TypedResult[O], error)
type DeliveryResult struct {
	ResponseRef    string
	SecretUpdates  map[string]string
	ResourceHealth *ResourceHealthReport
}

type DeliveryHandler[I any] func(context.Context, TypedRequest[I]) (DeliveryResult, error)

type BoundOperation struct {
	descriptor OperationDescriptor
	invoke     func(context.Context, CallRequest) (CallResult, error)
}

func (b BoundOperation) Descriptor() OperationDescriptor { return b.descriptor }

// BindCall performs the only type erasure for a synchronous operation.
func BindCall[I any, O any](operation CallOperation[I, O], handler CallHandler[I, O]) (BoundOperation, error) {
	return bindTypedOperation(operation.Descriptor(), handler)
}

func bindTypedOperation[I any, O any](descriptor OperationDescriptor, handler func(context.Context, TypedRequest[I]) (TypedResult[O], error)) (BoundOperation, error) {
	if err := descriptor.Validate(); err != nil {
		return BoundOperation{}, err
	}
	if handler == nil {
		return BoundOperation{}, fmt.Errorf("connector operation %s handler is required", descriptor.Key)
	}
	return BoundOperation{descriptor: descriptor, invoke: func(ctx context.Context, request CallRequest) (CallResult, error) {
		if err := validateInvocation(descriptor, request); err != nil {
			return CallResult{}, err
		}
		var input I
		if err := decodeStrict(request.Payload, &input); err != nil {
			return CallResult{}, fmt.Errorf("decode connector operation %s input: %w", descriptor.Key, err)
		}
		result, err := handler(ctx, TypedRequest[I]{
			Input: input, Connection: request.Connection, RequestRef: request.RequestRef,
			Headers: cloneStrings(request.Headers), Secrets: cloneStrings(request.Secrets),
			Delivery: request.Delivery, Principal: request.Principal,
		})
		if err != nil {
			return CallResult{
				ResponseRef: result.ResponseRef, SecretUpdates: cloneStrings(result.SecretUpdates), ResourceHealth: result.ResourceHealth,
			}, err
		}
		payload, err := json.Marshal(result.Output)
		if err != nil {
			return CallResult{}, fmt.Errorf("encode connector operation %s output: %w", descriptor.Key, err)
		}
		return CallResult{Payload: payload, ResponseRef: result.ResponseRef, SecretUpdates: cloneStrings(result.SecretUpdates), ResourceHealth: result.ResourceHealth}, nil
	}}, nil
}

// BindEnqueueDelivery erases the worker-side delivery handler for an enqueue
// operation. It does not provide a second Action-side execution channel.
func BindEnqueueDelivery[I any](operation EnqueueOperation[I], handler DeliveryHandler[I]) (BoundOperation, error) {
	return bindDeliveryOperation(operation.Descriptor(), handler)
}

// BindStartOperationDelivery erases the worker-side delivery handler that
// starts a long-running external operation after the local UoW commits.
func BindStartOperationDelivery[I any](operation StartOperation[I], handler DeliveryHandler[I]) (BoundOperation, error) {
	return bindDeliveryOperation(operation.Descriptor(), handler)
}

func bindDeliveryOperation[I any](descriptor OperationDescriptor, handler DeliveryHandler[I]) (BoundOperation, error) {
	if err := descriptor.Validate(); err != nil {
		return BoundOperation{}, err
	}
	if handler == nil {
		return BoundOperation{}, fmt.Errorf("connector operation %s handler is required", descriptor.Key)
	}
	return BoundOperation{descriptor: descriptor, invoke: func(ctx context.Context, request CallRequest) (CallResult, error) {
		if err := validateInvocation(descriptor, request); err != nil {
			return CallResult{}, err
		}
		var input I
		if err := decodeStrict(request.Payload, &input); err != nil {
			return CallResult{}, fmt.Errorf("decode connector operation %s input: %w", descriptor.Key, err)
		}
		result, err := handler(ctx, TypedRequest[I]{
			Input: input, Connection: request.Connection, RequestRef: request.RequestRef,
			Headers: cloneStrings(request.Headers), Secrets: cloneStrings(request.Secrets),
			Delivery: request.Delivery, Principal: request.Principal,
		})
		if err != nil {
			return CallResult{
				ResponseRef: result.ResponseRef, SecretUpdates: cloneStrings(result.SecretUpdates), ResourceHealth: result.ResourceHealth,
			}, err
		}
		return CallResult{ResponseRef: result.ResponseRef, SecretUpdates: cloneStrings(result.SecretUpdates), ResourceHealth: result.ResourceHealth}, nil
	}}, nil
}

func Call[I any, O any](ctx context.Context, gateway Gateway, operation CallOperation[I, O], input I) (O, error) {
	var zero O
	if gateway == nil {
		return zero, errors.New("connector gateway is required")
	}
	descriptor := operation.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return zero, err
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return zero, fmt.Errorf("encode connector operation %s input: %w", descriptor.Key, err)
	}
	result, err := gateway.Call(ctx, CallRequest{
		ConnectorKey: descriptor.ConnectorKey, ProviderKey: descriptor.ProviderKey,
		OperationKey: descriptor.Key, ContractSHA256: descriptor.ContractSHA256, Mode: descriptor.Mode, Payload: payload,
	})
	if err != nil {
		return zero, err
	}
	if err := decodeStrict(result.Payload, &zero); err != nil {
		return zero, fmt.Errorf("decode connector operation %s output: %w", descriptor.Key, err)
	}
	return zero, nil
}

func validateInvocation(expected OperationDescriptor, request CallRequest) error {
	if strings.TrimSpace(request.ConnectorKey) != expected.ConnectorKey || strings.TrimSpace(request.ProviderKey) != expected.ProviderKey {
		return fmt.Errorf("connector provider mismatch: got %s/%s, want %s/%s", request.ConnectorKey, request.ProviderKey, expected.ConnectorKey, expected.ProviderKey)
	}
	if strings.TrimSpace(request.OperationKey) != expected.Key {
		return fmt.Errorf("connector operation mismatch: got %q, want %q", request.OperationKey, expected.Key)
	}
	if request.Mode != expected.Mode {
		return fmt.Errorf("connector operation %s mode mismatch: got %q, want %q", expected.Key, request.Mode, expected.Mode)
	}
	if request.ContractSHA256 != expected.ContractSHA256 {
		return fmt.Errorf("connector operation %s contract hash mismatch", expected.Key)
	}
	return nil
}

func decodeStrict(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = json.RawMessage(`null`)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func cloneStrings(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
