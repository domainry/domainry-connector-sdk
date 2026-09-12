package connector

// OAuthOperationScopeProvider declares the actual OAuth grants required by one
// registered operation. Alternatives are ORed, members are ANDed. The absence
// of a declaration must not grant account access; a declared empty alternative
// explicitly requires no OAuth scope. This metadata neither authorizes the
// caller nor changes the operation's effect. Hosts must check both separately.
// Registry registration snapshots declarations for descriptor operations only.
type OAuthOperationScopeProvider interface {
	OAuthOperationScopes(operationKey string) (alternatives [][]string, declared bool)
}

func ResolveOAuthOperationScopes(adapter Adapter, operationKey string) ([][]string, bool) {
	if provider, ok := adapter.(OAuthOperationScopeProvider); ok {
		if values, declared := provider.OAuthOperationScopes(operationKey); declared {
			return cloneOAuthScopeAlternatives(values), true
		}
	}
	return nil, false
}
