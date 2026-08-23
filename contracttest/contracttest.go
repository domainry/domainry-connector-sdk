// Package contracttest provides reusable, side-effect-free checks for
// Connector implementations. It deliberately does not perform network I/O.
package contracttest

import (
	"errors"
	"fmt"

	"github.com/domainry/domainry-connector-sdk"
)

// ValidateAdapter checks the descriptor and the capability required by its
// declared reliability contracts.
func ValidateAdapter(adapter connector.Adapter) error {
	if adapter == nil {
		return errors.New("connector adapter is required")
	}
	descriptor := adapter.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return fmt.Errorf("validate provider descriptor: %w", err)
	}
	if descriptor.RequiresReconciler() {
		if _, ok := adapter.(connector.Reconciler); !ok {
			return fmt.Errorf("connector provider %s/%s declares reconciliation but does not implement Reconciler", descriptor.ConnectorKey, descriptor.ProviderKey)
		}
	}
	return nil
}
