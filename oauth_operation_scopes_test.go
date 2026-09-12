package connector

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type operationScopesFixture struct {
	Adapter
	scopes map[string][][]string
}

func (p *operationScopesFixture) OAuthOperationScopes(key string) ([][]string, bool) {
	v, ok := p.scopes[key]
	return v, ok
}

func TestOAuthOperationScopesSnapshotAndUnknownOperations(t *testing.T) {
	op := CallOperation[struct{}, struct{}]{ConnectorKey: "probe", ProviderKey: "scopes", Key: "lookup", ContractSHA256: strings.Repeat("a", 64), Reliability: ReliabilityContract{Effect: EffectRead, Idempotency: IdempotencyContract{Strategy: IdempotencyNatural}, Reconciliation: ReconciliationNone, Compensation: CompensationContract{Mode: CompensationNone}}}
	bound, err := BindCall(op, func(context.Context, TypedRequest[struct{}]) (TypedResult[struct{}], error) {
		return TypedResult[struct{}]{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewProvider(ProviderSchema{ConnectorKey: "probe", ProviderKey: "scopes", ProviderRevision: "1.0.0"}, bound)
	if err != nil {
		t.Fatal(err)
	}
	if _, declared := ResolveOAuthOperationScopes(base, "lookup"); declared {
		t.Fatal("legacy provider granted operation access")
	}
	for _, batch := range []bool{false, true} {
		source := &operationScopesFixture{Adapter: base, scopes: map[string][][]string{"lookup": {{"read", "profile"}, {"full"}}, "unregistered": {{}}}}
		registry := NewRegistry()
		if batch {
			err = registry.RegisterProviderSet(ProviderSet{Providers: []Adapter{source}})
		} else {
			err = registry.Register(source)
		}
		if err != nil {
			t.Fatal(err)
		}
		source.scopes["lookup"][0][0] = "mutated"
		delete(source.scopes, "lookup")
		registry.Freeze()
		frozen, _ := registry.Provider("probe", "scopes")
		want := [][]string{{"read", "profile"}, {"full"}}
		got, declared := ResolveOAuthOperationScopes(frozen, "lookup")
		if !declared || !reflect.DeepEqual(got, want) {
			t.Fatal("source mutation changed snapshot", got)
		}
		got[0][0] = "consumer-mutated"
		got, _ = ResolveOAuthOperationScopes(frozen, "lookup")
		if !reflect.DeepEqual(got, want) {
			t.Fatal("consumer mutation changed snapshot", got)
		}
		if _, declared := ResolveOAuthOperationScopes(frozen, "unregistered"); declared {
			t.Fatal("unregistered operation has a frozen declaration")
		}
	}
}

func TestOAuthOperationScopesPreservesDeclaredEmptyAndMissing(t *testing.T) {
	p := &operationScopesFixture{scopes: map[string][][]string{"invalid": nil, "no-scopes": {{}}}}
	if value, declared := ResolveOAuthOperationScopes(p, "invalid"); !declared || value != nil {
		t.Fatal(value, declared)
	}
	if value, declared := ResolveOAuthOperationScopes(p, "no-scopes"); !declared || len(value) != 1 || len(value[0]) != 0 {
		t.Fatal(value, declared)
	}
	if _, declared := ResolveOAuthOperationScopes(p, "missing"); declared {
		t.Fatal("missing declaration became authorization")
	}
}
