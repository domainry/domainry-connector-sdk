package contracttest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/domainry/domainry-connector-sdk"
	"github.com/domainry/domainry-connector-sdk/contracttest"
)

type adapter struct{ connector.Adapter }

func TestValidateAdapterRequiresDeclaredCapabilities(t *testing.T) {
	operation, err := connector.BindEnqueueDelivery(connector.EnqueueOperation[struct{}]{
		ConnectorKey: "example", ProviderKey: "test", Key: "send",
		ContractSHA256: strings.Repeat("a", 64),
		Reliability: connector.ReliabilityContract{
			Effect: connector.EffectWrite,
			Idempotency: connector.IdempotencyContract{
				Strategy:            connector.IdempotencyProviderKey,
				KeyRetentionSeconds: 60,
			},
			Reconciliation: connector.ReconciliationProviderLookup,
			Compensation:   connector.CompensationContract{Mode: connector.CompensationNone},
		},
	}, func(context.Context, connector.TypedRequest[struct{}]) (connector.DeliveryResult, error) {
		return connector.DeliveryResult{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := connector.NewProvider(connector.ProviderSchema{
		ConnectorKey: "example", ProviderKey: "test", ProviderRevision: "1",
	}, operation)
	if err != nil {
		t.Fatal(err)
	}
	if err := contracttest.ValidateAdapter(adapter{Adapter: provider}); err == nil || !strings.Contains(err.Error(), "does not implement Reconciler") {
		t.Fatalf("ValidateAdapter() error = %v", err)
	}
}
