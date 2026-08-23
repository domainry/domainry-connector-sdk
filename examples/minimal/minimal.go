// Package minimal demonstrates a Connector that performs no network access.
package minimal

import (
	"context"
	"strings"

	"github.com/domainry/domainry-connector-sdk"
)

type EchoInput struct {
	Message string `json:"message"`
}

type EchoOutput struct {
	Message string `json:"message"`
}

// Extensions constructs the example provider using only public SDK APIs.
func Providers(connector.Transport) (connector.ProviderSet, error) {
	bound, err := connector.BindCall(connector.CallOperation[EchoInput, EchoOutput]{
		ConnectorKey: "example", ProviderKey: "minimal", Key: "echo",
		ContractSHA256: strings.Repeat("a", 64),
		Reliability: connector.ReliabilityContract{
			Effect:         connector.EffectRead,
			Idempotency:    connector.IdempotencyContract{Strategy: connector.IdempotencyNatural},
			Reconciliation: connector.ReconciliationNone,
			Compensation:   connector.CompensationContract{Mode: connector.CompensationNone},
		},
	}, func(_ context.Context, request connector.TypedRequest[EchoInput]) (connector.TypedResult[EchoOutput], error) {
		return connector.TypedResult[EchoOutput]{Output: EchoOutput{Message: request.Input.Message}}, nil
	})
	if err != nil {
		return connector.ProviderSet{}, err
	}
	provider, err := connector.NewProvider(connector.ProviderSchema{
		ConnectorKey: "example", ProviderKey: "minimal", ProviderRevision: "1",
	}, bound)
	if err != nil {
		return connector.ProviderSet{}, err
	}
	return connector.ProviderSet{Providers: []connector.Adapter{provider}}, nil
}
