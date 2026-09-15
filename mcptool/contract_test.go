package mcptool

import (
	"strings"
	"testing"
)

func TestOperationContractsAndCallValidation(t *testing.T) {
	for key, expected := range map[string]string{
		TestConnectionOperationKey: TestConnectionOperationSHA256,
		ListToolsOperationKey:      ListToolsOperationSHA256,
		CallToolOperationKey:       CallToolOperationSHA256,
	} {
		if got := OperationSHA256(key); got != expected || len(got) != 64 {
			t.Fatalf("operation %s hash=%q", key, got)
		}
	}
	if OperationSHA256("unknown") != "" {
		t.Fatal("unknown MCP operation was accepted")
	}
	if err := (CallToolRequest{ToolName: "search", Arguments: map[string]any{"query": "agent"}}).Validate(); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []CallToolRequest{{}, {ToolName: " search"}, {ToolName: strings.Repeat("x", 129)}} {
		if invalid.Validate() == nil {
			t.Fatalf("invalid request accepted: %#v", invalid)
		}
	}
}
