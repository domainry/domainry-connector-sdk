package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// StreamDescriptor declares one provider-owned, continuously running input
// stream. Integration owns durable state, leases, fencing, retries, secrets,
// transports, and shutdown; the provider owns the remote protocol.
type StreamDescriptor struct {
	Key          string `json:"key"`
	StateVersion int    `json:"state_version"`
}

func (d StreamDescriptor) Validate() error {
	if strings.TrimSpace(d.Key) == "" || d.Key != strings.TrimSpace(d.Key) {
		return fmt.Errorf("connector stream key must be canonical")
	}
	if d.StateVersion <= 0 {
		return fmt.Errorf("connector stream %s requires a positive state version", d.Key)
	}
	return nil
}

// StreamRequest is the fenced snapshot supplied once for a stream session.
// A configuration or credential change cancels the session and starts a new
// one; providers must not retain this material after RunStream returns.
type StreamRequest struct {
	StreamKey    string            `json:"stream_key"`
	StateVersion int               `json:"state_version"`
	Connection   Connection        `json:"connection"`
	State        json.RawMessage   `json:"state,omitempty"`
	Secrets      map[string]string `json:"secrets,omitempty"`
	Now          time.Time         `json:"now"`
	Principal    Principal         `json:"principal"`
}

func (r StreamRequest) Validate() error {
	if strings.TrimSpace(r.StreamKey) == "" || r.StreamKey != strings.TrimSpace(r.StreamKey) {
		return fmt.Errorf("connector stream request requires a canonical stream key")
	}
	if r.StateVersion <= 0 {
		return fmt.Errorf("connector stream request requires a positive state version")
	}
	if strings.TrimSpace(r.Connection.ConnectorKey) == "" || strings.TrimSpace(r.Connection.ProviderKey) == "" || strings.TrimSpace(r.Connection.Key) == "" {
		return fmt.Errorf("connector stream request requires an exact connection identity")
	}
	if len(r.State) != 0 && !json.Valid(r.State) {
		return fmt.Errorf("connector stream request state must contain one JSON value")
	}
	if r.Now.IsZero() {
		return fmt.Errorf("connector stream request requires host time")
	}
	return nil
}

// StreamCheckpoint atomically advances provider-owned resumable state and
// accepts normalized events. Providers must wait for Commit to succeed before
// treating the remote sequence as durable.
type StreamCheckpoint struct {
	State    json.RawMessage     `json:"state"`
	Progress *BackgroundProgress `json:"progress,omitempty"`
	Events   []BackgroundEvent   `json:"events,omitempty"`
}

func (c StreamCheckpoint) Validate() error {
	if !json.Valid(c.State) {
		return fmt.Errorf("connector stream checkpoint state must contain one JSON value")
	}
	if c.Progress != nil {
		if err := c.Progress.Validate(); err != nil {
			return err
		}
	}
	for _, event := range c.Events {
		if err := event.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// StreamSink is serialized by Integration. Commit is rejected after the
// session loses its lease or generation fence.
type StreamSink interface {
	Commit(context.Context, StreamCheckpoint) error
}

// StreamProcessor is optional provider behavior. RunStream must honor context
// cancellation, return after the context is cancelled, and close every remote
// resource it opened. It may call the sink repeatedly for one session.
type StreamProcessor interface {
	Streams(Connection) []StreamDescriptor
	RunStream(context.Context, StreamRequest, StreamSink) error
}

// StreamCapabilityProvider is implemented by Registry snapshots. The boolean
// is false when the concrete Provider has no stream capability.
type StreamCapabilityProvider interface {
	StreamProcessor() (StreamProcessor, bool)
}
