package contracttest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/domainry/domainry-connector-sdk/connectorext"
	"github.com/domainry/domainry-connector-sdk/contracttest"
)

type adapter struct{ connectorext.Adapter }

func TestValidateAdapterRequiresDeclaredCapabilities(t *testing.T) {
	operation, err := connectorext.BindEnqueueDelivery(connectorext.EnqueueOperation[struct{}]{
		ConnectorKey: "example", ProviderKey: "test", Key: "send",
		ContractSHA256: strings.Repeat("a", 64),
		Reliability: connectorext.ReliabilityContract{
			Effect: connectorext.EffectWrite,
			Idempotency: connectorext.IdempotencyContract{
				Strategy:            connectorext.IdempotencyProviderKey,
				KeyRetentionSeconds: 60,
			},
			Reconciliation: connectorext.ReconciliationProviderLookup,
			Compensation:   connectorext.CompensationContract{Mode: connectorext.CompensationNone},
		},
	}, func(context.Context, connectorext.TypedRequest[struct{}]) (connectorext.DeliveryResult, error) {
		return connectorext.DeliveryResult{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := connectorext.NewProvider(connectorext.ProviderSchema{
		ConnectorKey: "example", ProviderKey: "test", ProviderRevision: "1",
	}, operation)
	if err != nil {
		t.Fatal(err)
	}
	if err := contracttest.ValidateAdapter(adapter{Adapter: provider}); err == nil || !strings.Contains(err.Error(), "does not implement Reconciler") {
		t.Fatalf("ValidateAdapter() error = %v", err)
	}
}
