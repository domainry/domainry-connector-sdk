package connector

import "encoding/json"

// ConnectorDefinition is one source-owned Connector product definition.
// Implementations may load it from embedded files, a signed release bundle or
// another deployment-owned source; consumers depend only on this SDK shape.
type ConnectorDefinition struct {
	Key     string
	Name    string
	Payload json.RawMessage
}

// DefinitionCatalog loads a detached Connector definition snapshot. Concrete
// catalog ownership stays at an outer composition root.
type DefinitionCatalog func() ([]ConnectorDefinition, error)
