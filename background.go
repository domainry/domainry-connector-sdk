package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BackgroundTaskDescriptor declares one provider-owned durable task. Integration
// owns scheduling, leases, persistence, secrets, transports, and audit; the
// provider owns the state payload and transition semantics.
type BackgroundTaskDescriptor struct {
	Key          string `json:"key"`
	StateVersion int    `json:"state_version"`
}

func (d BackgroundTaskDescriptor) Validate() error {
	if strings.TrimSpace(d.Key) == "" || d.Key != strings.TrimSpace(d.Key) {
		return fmt.Errorf("connector background task key must be canonical")
	}
	if d.StateVersion <= 0 {
		return fmt.Errorf("connector background task %s requires a positive state version", d.Key)
	}
	return nil
}

// BackgroundRequest is a single fenced transition request. State is opaque to
// Integration and must contain exactly one JSON value when non-empty.
type BackgroundRequest struct {
	TaskKey       string                     `json:"task_key"`
	StateVersion  int                        `json:"state_version"`
	Connection    Connection                 `json:"connection"`
	State         json.RawMessage            `json:"state,omitempty"`
	RelatedStates map[string]json.RawMessage `json:"related_states,omitempty"`
	Secrets       map[string]string          `json:"secrets,omitempty"`
	Now           time.Time                  `json:"now"`
	Principal     Principal                  `json:"principal"`
}

func (r BackgroundRequest) Validate() error {
	if strings.TrimSpace(r.TaskKey) == "" || r.TaskKey != strings.TrimSpace(r.TaskKey) {
		return fmt.Errorf("connector background request requires a canonical task key")
	}
	if r.StateVersion <= 0 {
		return fmt.Errorf("connector background request requires a positive state version")
	}
	if strings.TrimSpace(r.Connection.ConnectorKey) == "" || strings.TrimSpace(r.Connection.ProviderKey) == "" || strings.TrimSpace(r.Connection.Key) == "" {
		return fmt.Errorf("connector background request requires an exact connection identity")
	}
	if len(r.State) != 0 && !json.Valid(r.State) {
		return fmt.Errorf("connector background request state must contain one JSON value")
	}
	if r.Now.IsZero() {
		return fmt.Errorf("connector background request requires host time")
	}
	return nil
}

// BackgroundEvent is provider-normalized input that Integration must durably
// accept before executing any returned Commit operation.
type BackgroundEvent struct {
	ExternalID string          `json:"external_id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
}

func (e BackgroundEvent) Validate() error {
	if strings.TrimSpace(e.ExternalID) == "" || strings.TrimSpace(e.EventType) == "" {
		return fmt.Errorf("connector background event requires identity and type")
	}
	if !json.Valid(e.Payload) {
		return fmt.Errorf("connector background event payload must contain one JSON value")
	}
	return nil
}

// BackgroundCommit is a provider operation that Integration executes only after
// the state transition and events are durably committed. It is used for
// acknowledgements and other post-commit provider effects.
type BackgroundCommit struct {
	OperationKey   string          `json:"operation_key"`
	ContractSHA256 string          `json:"contract_sha256"`
	Payload        json.RawMessage `json:"payload"`
}

func (c BackgroundCommit) Validate() error {
	if strings.TrimSpace(c.OperationKey) == "" || strings.TrimSpace(c.ContractSHA256) == "" {
		return fmt.Errorf("connector background commit requires operation identity")
	}
	if !json.Valid(c.Payload) {
		return fmt.Errorf("connector background commit payload must contain one JSON value")
	}
	return nil
}

type BackgroundResult struct {
	State         json.RawMessage    `json:"state"`
	NextDueAt     time.Time          `json:"next_due_at,omitempty"`
	Events        []BackgroundEvent  `json:"events,omitempty"`
	Commit        []BackgroundCommit `json:"commit,omitempty"`
	WakeTasks     []string           `json:"wake_tasks,omitempty"`
	SecretUpdates map[string]string  `json:"secret_updates,omitempty"`
}

func (r BackgroundResult) Validate() error {
	if !json.Valid(r.State) {
		return fmt.Errorf("connector background result state must contain one JSON value")
	}
	for _, event := range r.Events {
		if err := event.Validate(); err != nil {
			return err
		}
	}
	for _, commit := range r.Commit {
		if err := commit.Validate(); err != nil {
			return err
		}
	}
	for _, task := range r.WakeTasks {
		if strings.TrimSpace(task) == "" || task != strings.TrimSpace(task) {
			return fmt.Errorf("connector background wake task must be canonical")
		}
	}
	return nil
}

// BackgroundProcessor is optional provider behavior. Implementations perform
// provider I/O only through the Transport supplied to their constructor.
type BackgroundProcessor interface {
	BackgroundTasks(Connection) []BackgroundTaskDescriptor
	ProcessBackground(context.Context, BackgroundRequest) (BackgroundResult, error)
}

// BackgroundCapabilityProvider is implemented by Registry snapshots. The
// boolean is false when the concrete Provider has no background capability.
type BackgroundCapabilityProvider interface {
	BackgroundProcessor() (BackgroundProcessor, bool)
}

// BackgroundCleanupProcessor lets a provider tear down remote resources before
// Integration deletes a connection. Integration still resolves and persists secrets.
type BackgroundCleanupProcessor interface {
	CleanupBackground(context.Context, Connection, map[string]string, time.Time, Principal) (map[string]string, error)
}

type BackgroundCleanupCapabilityProvider interface {
	BackgroundCleanupProcessor() (BackgroundCleanupProcessor, bool)
}
