package connector

import "context"

// OAuthConnectionTestScopeProvider declares the OAuth scopes needed by the
// provider's fixed connection probe. Alternatives are ORed; every scope in an
// alternative is required. A missing declaration is distinct from a declared
// empty alternative (no scopes required). This never adds requested scopes or
// grants business access. Registry registration snapshots this metadata.
type OAuthConnectionTestScopeProvider interface {
	OAuthConnectionTestScopes() (alternatives [][]string, declared bool)
}

func ResolveOAuthConnectionTestScopes(adapter Adapter) ([][]string, bool) {
	if provider, ok := adapter.(OAuthConnectionTestScopeProvider); ok {
		values, declared := provider.OAuthConnectionTestScopes()
		if declared {
			return cloneOAuthScopeAlternatives(values), true
		}
	}
	return nil, false
}

func cloneOAuthScopeAlternatives(values [][]string) [][]string {
	if values == nil {
		return nil
	}
	out := make([][]string, len(values))
	for i := range values {
		out[i] = append([]string(nil), values[i]...)
	}
	return out
}

// OAuthAuthorizer is an optional account-authorization protocol capability, not
// a model-callable business operation. Integration owns the session, current-user
// authorization, exact redirect allowlist, state, PKCE, one-time code consumption,
// encrypted token persistence and recovery. Providers perform protocol I/O only
// through the host Transport. Hosts must never blindly retry an uncertain exchange.
type OAuthAuthorizer interface {
	AuthorizationURL(OAuthAuthorizationRequest) (string, error)
	ExchangeAuthorizationCode(context.Context, OAuthCodeExchangeRequest) (OAuthTokens, error)
}

// OAuthAuthorizationCapabilityProvider preserves the optional protocol capability
// when a host freezes a registry, without advertising it for non-OAuth providers.
type OAuthAuthorizationCapabilityProvider interface {
	OAuthAuthorizer() (OAuthAuthorizer, bool)
}

func ResolveOAuthAuthorizer(adapter Adapter) (OAuthAuthorizer, bool) {
	if authorizer, ok := adapter.(OAuthAuthorizer); ok {
		return authorizer, true
	}
	if capabilities, ok := adapter.(OAuthAuthorizationCapabilityProvider); ok {
		return capabilities.OAuthAuthorizer()
	}
	return nil, false
}

type OAuthAuthorizationRequest struct {
	Connection  Connection `json:"connection"`
	ClientID    string     `json:"client_id"`
	RedirectURI string     `json:"redirect_uri"`
	Scopes      []string   `json:"scopes"`
	// State and CodeChallenge come from fresh cryptographic randomness generated
	// and durably bound to the authenticated session by Integration.
	State         string `json:"-"`
	CodeChallenge string `json:"code_challenge"`
}

type OAuthCodeExchangeRequest struct {
	Connection      Connection `json:"connection"`
	ClientID        string     `json:"-"`
	ClientSecret    string     `json:"-"`
	RedirectURI     string     `json:"redirect_uri"`
	RequestedScopes []string   `json:"requested_scopes"`
	Code            string     `json:"-"`
	CodeVerifier    string     `json:"-"`
}

// OAuthTokens is private Provider-to-Integration output. Never serialize raw
// credentials into traces, model payloads or product responses. GrantedScopes
// record the actual response (or requested scope when omitted under RFC 6749),
// not product permission: hosts still intersect their current grants and policy.
type OAuthTokens struct {
	AccessToken      string   `json:"-"`
	RefreshToken     string   `json:"-"`
	TokenType        string   `json:"token_type"`
	GrantedScopes    []string `json:"granted_scopes"`
	ExpiresInSeconds int64    `json:"expires_in_seconds"`
}
