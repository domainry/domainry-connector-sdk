package connector

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

type oauthTestProvider struct{ Adapter }
type oauthScopeTestProvider struct {
	Adapter
	scopes [][]string
}

func (p oauthScopeTestProvider) OAuthConnectionTestScopes() ([][]string, bool) { return p.scopes, true }

func (oauthTestProvider) AuthorizationURL(OAuthAuthorizationRequest) (string, error) {
	return "https://example.test/authorize", nil
}
func (oauthTestProvider) ExchangeAuthorizationCode(context.Context, OAuthCodeExchangeRequest) (OAuthTokens, error) {
	return OAuthTokens{AccessToken: "private-access", RefreshToken: "private-refresh", TokenType: "Bearer"}, nil
}

func TestOAuthOptionalCapabilitySurvivesRegistryAndSecretsAreNotSerialized(t *testing.T) {
	operation := CallOperation[struct{}, struct{}]{ConnectorKey: "probe", ProviderKey: "oauth", Key: "lookup", ContractSHA256: strings.Repeat("a", 64), Reliability: ReliabilityContract{Effect: EffectRead, Idempotency: IdempotencyContract{Strategy: IdempotencyNatural}, Reconciliation: ReconciliationNone, Compensation: CompensationContract{Mode: CompensationNone}}}
	bound, err := BindCall(operation, func(context.Context, TypedRequest[struct{}]) (TypedResult[struct{}], error) {
		return TypedResult[struct{}]{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewProvider(ProviderSchema{ConnectorKey: "probe", ProviderKey: "oauth", ProviderRevision: "1.0.0"}, bound)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ResolveOAuthAuthorizer(base); ok {
		t.Fatal("non-OAuth provider advertised authorization")
	}
	if _, declared := ResolveOAuthConnectionTestScopes(base); declared {
		t.Fatal("legacy provider declares scope requirements")
	}
	for _, batch := range []bool{false, true} {
		registry := NewRegistry()
		if batch {
			err = registry.RegisterProviderSet(ProviderSet{Providers: []Adapter{oauthTestProvider{base}}})
		} else {
			err = registry.Register(oauthTestProvider{base})
		}
		if err != nil {
			t.Fatal(err)
		}
		registry.Freeze()
		if frozen, _ := registry.Provider("probe", "oauth"); frozen != nil {
			if _, declared := ResolveOAuthConnectionTestScopes(frozen); declared {
				t.Fatal("wrapper invented scope declaration")
			}
		}
		frozen, _ := registry.Provider("probe", "oauth")
		authorizer, ok := ResolveOAuthAuthorizer(frozen)
		if !ok {
			t.Fatal("registry discarded optional OAuth")
		}
		tokens, err := authorizer.ExchangeAuthorizationCode(t.Context(), OAuthCodeExchangeRequest{})
		if err != nil || tokens.AccessToken != "private-access" {
			t.Fatal("frozen OAuth delegation failed", err)
		}
		for _, value := range []any{tokens, OAuthAuthorizationRequest{State: "private-state"}, OAuthCodeExchangeRequest{ClientID: "private-client-id", ClientSecret: "private-client-secret", Code: "private-code", CodeVerifier: "private-verifier"}} {
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "private-") {
				t.Fatalf("OAuth serialization leaked private fields: %s", raw)
			}
		}
	}
}

func TestOAuthTestScopesRegistrySnapshot(t *testing.T) {
	operation := CallOperation[struct{}, struct{}]{ConnectorKey: "probe", ProviderKey: "scopes", Key: "lookup", ContractSHA256: strings.Repeat("a", 64), Reliability: ReliabilityContract{Effect: EffectRead, Idempotency: IdempotencyContract{Strategy: IdempotencyNatural}, Reconciliation: ReconciliationNone, Compensation: CompensationContract{Mode: CompensationNone}}}
	bound, err := BindCall(operation, func(context.Context, TypedRequest[struct{}]) (TypedResult[struct{}], error) {
		return TypedResult[struct{}]{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewProvider(ProviderSchema{ConnectorKey: "probe", ProviderKey: "scopes", ProviderRevision: "1.0.0"}, bound)
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range []bool{false, true} {
		source := [][]string{{"profile", "email"}, {"openid"}}
		provider := oauthScopeTestProvider{Adapter: base, scopes: source}
		r := NewRegistry()
		if batch {
			err = r.RegisterProviderSet(ProviderSet{Providers: []Adapter{provider}})
		} else {
			err = r.Register(provider)
		}
		if err != nil {
			t.Fatal(err)
		}
		source[0][0] = "write"
		source[1] = nil
		r.Freeze()
		frozen, _ := r.Provider("probe", "scopes")
		first, declared := ResolveOAuthConnectionTestScopes(frozen)
		want := [][]string{{"profile", "email"}, {"openid"}}
		if !declared || !reflect.DeepEqual(first, want) {
			t.Fatalf("source mutated registered metadata: %v", first)
		}
		first[0][0] = "write"
		first[1] = nil
		second, _ := ResolveOAuthConnectionTestScopes(frozen)
		if !reflect.DeepEqual(second, want) {
			t.Fatalf("consumer mutated registered metadata: %v", second)
		}
	}
}
