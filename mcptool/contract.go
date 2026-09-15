// Package mcptool owns the public Connector operation contract used to list
// and invoke tools on an MCP server. Connection configuration and credentials
// remain private to Integration and Connectors.
package mcptool

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

const (
	ConnectorKey = "mcp_tool"
	ProviderKey  = "mcp"

	TestConnectionOperationKey    = "test_connection"
	TestConnectionOperationSHA256 = "2f4a1bc6298cb8d22d02956dc2c1992cc24354638153e07967015bc4d592c9f3"
	ListToolsOperationKey         = "list_tools"
	ListToolsOperationSHA256      = "80912e3d2921f06343a653cb32559bad4b1085bb86f793599ae6da341f4df929"
	CallToolOperationKey          = "call_tool"
	CallToolOperationSHA256       = "1a88ffb95d28a14734d2c5a417da8a6626a94b64f37a0ffa5051fbe7abf5beba"
)

func OperationSHA256(key string) string {
	switch key {
	case TestConnectionOperationKey:
		return TestConnectionOperationSHA256
	case ListToolsOperationKey:
		return ListToolsOperationSHA256
	case CallToolOperationKey:
		return CallToolOperationSHA256
	default:
		return ""
	}
}

type ListToolsRequest struct{}

type CallToolRequest struct {
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Approved  bool           `json:"approved,omitempty"`
}

func (r CallToolRequest) Validate() error {
	if !boundedText(r.ToolName, 128) {
		return fmt.Errorf("MCP tool name is invalid")
	}
	if r.Arguments == nil {
		r.Arguments = map[string]any{}
	}
	raw, err := json.Marshal(r.Arguments)
	if err != nil || len(raw) > 1<<20 {
		return fmt.Errorf("MCP tool arguments are invalid")
	}
	return nil
}

func boundedText(value string, limit int) bool {
	return value != "" && len(value) <= limit && strings.TrimSpace(value) == value && strings.IndexFunc(value, unicode.IsControl) < 0
}
