package connectorext_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConnectorextCompilesFromOutsideTheRepositoryModule(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
	externalRoot := t.TempDir()
	goMod := []byte("module example.com/connector-project\n\ngo 1.26.0\n\nrequire github.com/domainry/domainry-connector-sdk v0.0.0\n\nreplace github.com/domainry/domainry-connector-sdk => " + repositoryRoot + "\n")
	if err := os.WriteFile(filepath.Join(externalRoot, "go.mod"), goMod, 0o600); err != nil {
		t.Fatal(err)
	}
	source := []byte(`package connector

import (
	"context"

	"github.com/domainry/domainry-connector-sdk/connectorext"
)

type Request struct { MemberID string ` + "`json:\"member_id\"`" + ` }
type Response struct { Name string ` + "`json:\"name\"`" + ` }

type ProjectProvider struct { connectorext.Adapter }

func (ProjectProvider) ValidateConfig(connectorext.Connection) error { return nil }

func (ProjectProvider) TestConnection(context.Context, connectorext.TestConnectionRequest) (connectorext.TestConnectionResult, error) {
	return connectorext.TestConnectionResult{Connected: true}, nil
}

func (ProjectProvider) VerifyWebhook(context.Context, connectorext.VerifyWebhookRequest) (connectorext.VerifiedWebhook, error) {
	return connectorext.VerifiedWebhook{EventType: "member.updated", ExternalID: "event-1", Payload: []byte(` + "`{\"member_id\":\"member-1\"}`" + `)}, nil
}

func (ProjectProvider) Reconcile(context.Context, connectorext.ReconcileRequest) (connectorext.ReconcileResult, error) {
	return connectorext.ReconcileResult{Outcome: connectorext.ReconciliationSucceeded}, nil
}

var _ connectorext.ConfigValidator = ProjectProvider{}
var _ connectorext.ConnectionTester = ProjectProvider{}
var _ connectorext.WebhookVerifier = ProjectProvider{}
var _ connectorext.Reconciler = ProjectProvider{}

func classifiedFailures(cause error) []error {
	return []error{
		connectorext.RetryableError("acme.rate_limited", cause),
		connectorext.PermanentError("acme.request_rejected", cause),
		connectorext.UncertainError("acme.delivery_unknown", cause),
	}
}

var GetMember = connectorext.CallOperation[Request, Response]{
	ConnectorKey: "member_center", ProviderKey: "acme",
	Key: "get_member", ContractSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	Reliability: connectorext.ReliabilityContract{Effect: connectorext.EffectRead, Idempotency: connectorext.IdempotencyContract{Strategy: connectorext.IdempotencyNatural}, Reconciliation: connectorext.ReconciliationNone, Compensation: connectorext.CompensationContract{Mode: connectorext.CompensationNone}},
}
var SendNotice = connectorext.EnqueueOperation[Request]{
	ConnectorKey: "member_center", ProviderKey: "acme",
	Key: "send_notice", ContractSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	Reliability: connectorext.ReliabilityContract{Effect: connectorext.EffectWrite, Idempotency: connectorext.IdempotencyContract{Strategy: connectorext.IdempotencyProviderKey, KeyRetentionSeconds: 86400}, Reconciliation: connectorext.ReconciliationProviderLookup, Compensation: connectorext.CompensationContract{Mode: connectorext.CompensationNone}},
}
var SyncMember = connectorext.StartOperation[Request]{
	ConnectorKey: "member_center", ProviderKey: "acme",
	Key: "sync_member", ContractSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	Reliability: connectorext.ReliabilityContract{Effect: connectorext.EffectWrite, Idempotency: connectorext.IdempotencyContract{Strategy: connectorext.IdempotencyProviderKey, KeyRetentionSeconds: 86400}, Reconciliation: connectorext.ReconciliationProviderLookup, Compensation: connectorext.CompensationContract{Mode: connectorext.CompensationNone}},
}

func Extensions() (connectorext.ExtensionSet, error) {
	call, err := connectorext.BindCall(GetMember, func(context.Context, connectorext.TypedRequest[Request]) (connectorext.TypedResult[Response], error) {
		return connectorext.TypedResult[Response]{Output: Response{Name: "Ada"}}, nil
	})
	if err != nil { return connectorext.ExtensionSet{}, err }
	enqueue, err := connectorext.BindEnqueueDelivery(SendNotice, func(context.Context, connectorext.TypedRequest[Request]) (connectorext.DeliveryResult, error) {
		return connectorext.DeliveryResult{ResponseRef: "delivery-1"}, nil
	})
	if err != nil { return connectorext.ExtensionSet{}, err }
	start, err := connectorext.BindStartOperationDelivery(SyncMember, func(context.Context, connectorext.TypedRequest[Request]) (connectorext.DeliveryResult, error) {
		return connectorext.DeliveryResult{ResponseRef: "remote-operation-1"}, nil
	})
	if err != nil { return connectorext.ExtensionSet{}, err }
	provider, err := connectorext.NewProvider(connectorext.ProviderSchema{
		ConnectorKey: "member_center", ProviderKey: "acme", ProviderRevision: "provider-v1",
		ConfigFields: []connectorext.ConfigField{{Key: "region", Name: "Region", Type: connectorext.ConfigFieldSelect, Default: []byte(` + "`\"us\"`" + `), Validation: connectorext.ConfigValidation{Options: []string{"us", "eu"}}}},
		SecretFields: []connectorext.SecretField{{Key: "api_key", Name: "API key", Required: true, CredentialKind: connectorext.SecretCredentialAPIKey, MaterialFormat: connectorext.SecretMaterialOpaque, RotationPolicy: connectorext.SecretRotationManual, ExpiryPolicy: connectorext.SecretExpiryOptional, TestRequirement: connectorext.SecretTestWhenBound}},
	}, call, enqueue, start)
	if err != nil { return connectorext.ExtensionSet{}, err }
	return connectorext.ExtensionSet{Providers: []connectorext.Adapter{ProjectProvider{Adapter: provider}}}, nil
}
`)
	if err := os.WriteFile(filepath.Join(externalRoot, "connector.go"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = externalRoot
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("external project cannot compile connectorext: %v\n%s", err, output)
	}

	invalid := []byte(`package connector

import (
	"context"

	"github.com/domainry/domainry-connector-sdk/connectorext"
)

func invalid(ctx context.Context, gateway connectorext.Gateway) {
	_, _ = connectorext.Call(ctx, gateway, SendNotice, Request{MemberID: "member-1"})
}
`)
	if err := os.WriteFile(filepath.Join(externalRoot, "invalid.go"), invalid, 0o600); err != nil {
		t.Fatal(err)
	}
	command = exec.Command("go", "test", "./...")
	command.Dir = externalRoot
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("enqueue operation compiled through the synchronous Call API")
	}
	if !strings.Contains(string(output), "EnqueueOperation") || !strings.Contains(string(output), "CallOperation") {
		t.Fatalf("external compile failure did not prove operation kind separation: %v\n%s", err, output)
	}

	noInvoker := []byte(`package connector

import "github.com/domainry/domainry-connector-sdk/connectorext"

var _ connectorext.Invoker
`)
	if err := os.WriteFile(filepath.Join(externalRoot, "invalid.go"), noInvoker, 0o600); err != nil {
		t.Fatal(err)
	}
	command = exec.Command("go", "test", "./...")
	command.Dir = externalRoot
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err = command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "undefined: connectorext.Invoker") {
		t.Fatalf("public optional Invoker unexpectedly compiled: %v\n%s", err, output)
	}
}
