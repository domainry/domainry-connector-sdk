package connector

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type backgroundAdapter struct{ *mutableAdapter }

func (*backgroundAdapter) BackgroundTasks(Connection) []BackgroundTaskDescriptor {
	return []BackgroundTaskDescriptor{{Key: "sync", StateVersion: 1}}
}

func (*backgroundAdapter) ProcessBackground(context.Context, BackgroundRequest) (BackgroundResult, error) {
	return BackgroundResult{State: json.RawMessage(`{"cursor":"2"}`)}, nil
}

func TestBackgroundContractsAndFrozenRegistryCapability(t *testing.T) {
	request := BackgroundRequest{TaskKey: "sync", StateVersion: 1, Connection: Connection{Key: "primary", ConnectorKey: "mail", ProviderKey: "provider"}, State: json.RawMessage(`{"cursor":"1"}`), Now: time.Now().UTC()}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (BackgroundTaskDescriptor{Key: "sync", StateVersion: 1}).Validate(); err != nil {
		t.Fatal(err)
	}
	descriptor := memberProvider(t).Descriptor()
	provider := &backgroundAdapter{mutableAdapter: &mutableAdapter{descriptor: descriptor}}
	registry := NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatal(err)
	}
	registry.Freeze()
	resolved, ok := registry.Provider(descriptor.ConnectorKey, descriptor.ProviderKey)
	if !ok {
		t.Fatal("provider is missing")
	}
	capability, ok := resolved.(BackgroundCapabilityProvider)
	if !ok {
		t.Fatalf("frozen provider lost background capability accessor: %T", resolved)
	}
	processor, ok := capability.BackgroundProcessor()
	if !ok {
		t.Fatal("background processor is missing")
	}
	result, err := processor.ProcessBackground(t.Context(), request)
	if err != nil || string(result.State) != `{"cursor":"2"}` {
		t.Fatalf("result=%s error=%v", result.State, err)
	}
}
