// Package minimal demonstrates a Connector that performs no network access.
package minimal

import (
	"context"
	"strings"

	"github.com/domainry/domainry-connector-sdk/connectorext"
)

type EchoInput struct {
	Message string `json:"message"`
}

type EchoOutput struct {
	Message string `json:"message"`
}

// Extensions constructs the example provider using only public SDK APIs.
func Extensions(connectorext.Transport) (connectorext.ExtensionSet, error) {
	bound, err := connectorext.BindCall(connectorext.CallOperation[EchoInput, EchoOutput]{
		ConnectorKey: "example", ProviderKey: "minimal", Key: "echo",
		ContractSHA256: strings.Repeat("a", 64),
		Reliability: connectorext.ReliabilityContract{
			Effect:         connectorext.EffectRead,
			Idempotency:    connectorext.IdempotencyContract{Strategy: connectorext.IdempotencyNatural},
			Reconciliation: connectorext.ReconciliationNone,
			Compensation:   connectorext.CompensationContract{Mode: connectorext.CompensationNone},
		},
	}, func(_ context.Context, request connectorext.TypedRequest[EchoInput]) (connectorext.TypedResult[EchoOutput], error) {
		return connectorext.TypedResult[EchoOutput]{Output: EchoOutput{Message: request.Input.Message}}, nil
	})
	if err != nil {
		return connectorext.ExtensionSet{}, err
	}
	provider, err := connectorext.NewProvider(connectorext.ProviderSchema{
		ConnectorKey: "example", ProviderKey: "minimal", ProviderRevision: "1",
	}, bound)
	if err != nil {
		return connectorext.ExtensionSet{}, err
	}
	return connectorext.ExtensionSet{Providers: []connectorext.Adapter{provider}}, nil
}
