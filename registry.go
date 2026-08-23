package connector

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	ErrRegistryFrozen    = errors.New("connector provider registry is frozen")
	ErrProviderRequired  = errors.New("connector provider adapter is required")
	ErrProviderDuplicate = errors.New("connector provider is already registered")
)

// ProviderSet is one explicit batch of Provider Adapters composed by a project
// before the Runtime validates and freezes its Registry.
type ProviderSet struct {
	Providers []Adapter
}

type Registry struct {
	mu        sync.RWMutex
	frozen    bool
	providers map[string]Adapter
}

type frozenAdapter struct {
	descriptor ProviderDescriptor
	delegate   Adapter
}

type frozenConfigValidator struct {
	*frozenAdapter
	ConfigValidator
}

type frozenConnectionTester struct {
	*frozenAdapter
	ConnectionTester
}

type frozenConfigValidatorAndConnectionTester struct {
	*frozenAdapter
	ConfigValidator
	ConnectionTester
}

type frozenWebhookVerifier struct {
	*frozenAdapter
	WebhookVerifier
}

type frozenConfigValidatorAndWebhookVerifier struct {
	*frozenAdapter
	ConfigValidator
	WebhookVerifier
}

type frozenConnectionTesterAndWebhookVerifier struct {
	*frozenAdapter
	ConnectionTester
	WebhookVerifier
}

type frozenConfigValidatorConnectionTesterAndWebhookVerifier struct {
	*frozenAdapter
	ConfigValidator
	ConnectionTester
	WebhookVerifier
}

type frozenReconciler struct {
	*frozenAdapter
	Reconciler
}

type frozenConfigValidatorAndReconciler struct {
	*frozenAdapter
	ConfigValidator
	Reconciler
}

type frozenConnectionTesterAndReconciler struct {
	*frozenAdapter
	ConnectionTester
	Reconciler
}

type frozenConfigValidatorConnectionTesterAndReconciler struct {
	*frozenAdapter
	ConfigValidator
	ConnectionTester
	Reconciler
}

type frozenWebhookVerifierAndReconciler struct {
	*frozenAdapter
	WebhookVerifier
	Reconciler
}

type frozenConfigValidatorWebhookVerifierAndReconciler struct {
	*frozenAdapter
	ConfigValidator
	WebhookVerifier
	Reconciler
}

type frozenAllOptionalCapabilities struct {
	*frozenAdapter
	ConnectionTester
	WebhookVerifier
	Reconciler
}

type frozenAllCapabilities struct {
	*frozenAdapter
	ConfigValidator
	ConnectionTester
	WebhookVerifier
	Reconciler
}

func (a *frozenAdapter) Descriptor() ProviderDescriptor {
	return cloneProviderDescriptor(a.descriptor)
}

func (a *frozenAdapter) Call(ctx context.Context, request CallRequest) (CallResult, error) {
	return a.delegate.Call(ctx, request)
}

func freezeAdapter(provider Adapter, descriptor ProviderDescriptor) Adapter {
	base := &frozenAdapter{descriptor: cloneProviderDescriptor(descriptor), delegate: provider}
	validator, hasValidator := provider.(ConfigValidator)
	tester, hasTester := provider.(ConnectionTester)
	verifier, hasVerifier := provider.(WebhookVerifier)
	reconciler, hasReconciler := provider.(Reconciler)
	switch {
	case hasValidator && hasTester && hasVerifier && hasReconciler:
		return &frozenAllCapabilities{frozenAdapter: base, ConfigValidator: validator, ConnectionTester: tester, WebhookVerifier: verifier, Reconciler: reconciler}
	case hasValidator && hasTester && hasVerifier:
		return &frozenConfigValidatorConnectionTesterAndWebhookVerifier{frozenAdapter: base, ConfigValidator: validator, ConnectionTester: tester, WebhookVerifier: verifier}
	case hasValidator && hasTester && hasReconciler:
		return &frozenConfigValidatorConnectionTesterAndReconciler{frozenAdapter: base, ConfigValidator: validator, ConnectionTester: tester, Reconciler: reconciler}
	case hasValidator && hasVerifier && hasReconciler:
		return &frozenConfigValidatorWebhookVerifierAndReconciler{frozenAdapter: base, ConfigValidator: validator, WebhookVerifier: verifier, Reconciler: reconciler}
	case hasValidator && hasTester:
		return &frozenConfigValidatorAndConnectionTester{frozenAdapter: base, ConfigValidator: validator, ConnectionTester: tester}
	case hasValidator && hasVerifier:
		return &frozenConfigValidatorAndWebhookVerifier{frozenAdapter: base, ConfigValidator: validator, WebhookVerifier: verifier}
	case hasValidator && hasReconciler:
		return &frozenConfigValidatorAndReconciler{frozenAdapter: base, ConfigValidator: validator, Reconciler: reconciler}
	case hasValidator:
		return &frozenConfigValidator{frozenAdapter: base, ConfigValidator: validator}
	case hasTester && hasVerifier && hasReconciler:
		return &frozenAllOptionalCapabilities{frozenAdapter: base, ConnectionTester: tester, WebhookVerifier: verifier, Reconciler: reconciler}
	case hasTester && hasVerifier:
		return &frozenConnectionTesterAndWebhookVerifier{frozenAdapter: base, ConnectionTester: tester, WebhookVerifier: verifier}
	case hasTester && hasReconciler:
		return &frozenConnectionTesterAndReconciler{frozenAdapter: base, ConnectionTester: tester, Reconciler: reconciler}
	case hasVerifier && hasReconciler:
		return &frozenWebhookVerifierAndReconciler{frozenAdapter: base, WebhookVerifier: verifier, Reconciler: reconciler}
	case hasTester:
		return &frozenConnectionTester{frozenAdapter: base, ConnectionTester: tester}
	case hasVerifier:
		return &frozenWebhookVerifier{frozenAdapter: base, WebhookVerifier: verifier}
	case hasReconciler:
		return &frozenReconciler{frozenAdapter: base, Reconciler: reconciler}
	default:
		return base
	}
}

func NewRegistry() *Registry { return &Registry{providers: map[string]Adapter{}} }

func ProviderIdentity(connectorKey, providerKey string) string {
	return strings.TrimSpace(connectorKey) + ":" + strings.TrimSpace(providerKey)
}

func (r *Registry) Register(provider Adapter) error {
	if provider == nil {
		return ErrProviderRequired
	}
	descriptor := provider.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return err
	}
	if descriptor.RequiresReconciler() {
		if _, ok := provider.(Reconciler); !ok {
			return fmt.Errorf("connector provider %s/%s declares provider reconciliation but does not implement Reconciler", descriptor.ConnectorKey, descriptor.ProviderKey)
		}
	}
	key := ProviderIdentity(descriptor.ConnectorKey, descriptor.ProviderKey)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		return ErrRegistryFrozen
	}
	if _, exists := r.providers[key]; exists {
		return fmt.Errorf("%w: %s", ErrProviderDuplicate, key)
	}
	r.providers[key] = freezeAdapter(provider, descriptor)
	return nil
}

func (r *Registry) RegisterProviderSet(set ProviderSet) error {
	validated := make(map[string]Adapter, len(set.Providers))
	for _, provider := range set.Providers {
		if provider == nil {
			return ErrProviderRequired
		}
		descriptor := provider.Descriptor()
		if err := descriptor.Validate(); err != nil {
			return err
		}
		if descriptor.RequiresReconciler() {
			if _, ok := provider.(Reconciler); !ok {
				return fmt.Errorf("connector provider %s/%s declares provider reconciliation but does not implement Reconciler", descriptor.ConnectorKey, descriptor.ProviderKey)
			}
		}
		key := ProviderIdentity(descriptor.ConnectorKey, descriptor.ProviderKey)
		if _, exists := validated[key]; exists {
			return fmt.Errorf("%w: %s", ErrProviderDuplicate, key)
		}
		validated[key] = freezeAdapter(provider, descriptor)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		return ErrRegistryFrozen
	}
	for key := range validated {
		if _, exists := r.providers[key]; exists {
			return fmt.Errorf("%w: %s", ErrProviderDuplicate, key)
		}
	}
	for key, provider := range validated {
		r.providers[key] = provider
	}
	return nil
}

func (r *Registry) Freeze() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frozen = true
}

func (r *Registry) Frozen() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.frozen
}

func (r *Registry) Provider(connectorKey, providerKey string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[ProviderIdentity(connectorKey, providerKey)]
	return provider, ok
}

func (r *Registry) Descriptors() []ProviderDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ProviderDescriptor, 0, len(r.providers))
	for _, provider := range r.providers {
		result = append(result, provider.Descriptor())
	}
	sort.Slice(result, func(i, j int) bool {
		return ProviderIdentity(result[i].ConnectorKey, result[i].ProviderKey) < ProviderIdentity(result[j].ConnectorKey, result[j].ProviderKey)
	})
	return result
}

func (r *Registry) Providers() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.providers))
	for key := range r.providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]Adapter, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.providers[key])
	}
	return result
}
