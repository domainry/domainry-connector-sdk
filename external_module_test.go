package connector_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConnectorCompilesFromOutsideTheRepositoryModule(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	repositoryRoot := filepath.Dir(currentFile)
	externalRoot := t.TempDir()
	goMod := []byte("module example.com/connector-project\n\ngo 1.26.0\n\nrequire github.com/domainry/domainry-connector-sdk v0.0.0\n\nreplace github.com/domainry/domainry-connector-sdk => " + repositoryRoot + "\n")
	if err := os.WriteFile(filepath.Join(externalRoot, "go.mod"), goMod, 0o600); err != nil {
		t.Fatal(err)
	}
	source := []byte(`package connector

import (
	"context"

	"github.com/domainry/domainry-connector-sdk"
)

type Request struct { MemberID string ` + "`json:\"member_id\"`" + ` }
type Response struct { Name string ` + "`json:\"name\"`" + ` }

type ProjectProvider struct { connector.Adapter }

func (ProjectProvider) ValidateConfig(connector.Connection) error { return nil }

func (ProjectProvider) TestConnection(context.Context, connector.TestConnectionRequest) (connector.TestConnectionResult, error) {
	return connector.TestConnectionResult{Connected: true}, nil
}

func (ProjectProvider) VerifyWebhook(context.Context, connector.VerifyWebhookRequest) (connector.VerifiedWebhook, error) {
	return connector.VerifiedWebhook{EventType: "member.updated", ExternalID: "event-1", Payload: []byte(` + "`{\"member_id\":\"member-1\"}`" + `)}, nil
}

func (ProjectProvider) Reconcile(context.Context, connector.ReconcileRequest) (connector.ReconcileResult, error) {
	return connector.ReconcileResult{Outcome: connector.ReconciliationSucceeded}, nil
}

var _ connector.ConfigValidator = ProjectProvider{}
var _ connector.ConnectionTester = ProjectProvider{}
var _ connector.WebhookVerifier = ProjectProvider{}
var _ connector.Reconciler = ProjectProvider{}

func classifiedFailures(cause error) []error {
	return []error{
		connector.RetryableError("acme.rate_limited", cause),
		connector.PermanentError("acme.request_rejected", cause),
		connector.UncertainError("acme.delivery_unknown", cause),
	}
}

var GetMember = connector.CallOperation[Request, Response]{
	ConnectorKey: "member_center", ProviderKey: "acme",
	Key: "get_member", ContractSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	Reliability: connector.ReliabilityContract{Effect: connector.EffectRead, Idempotency: connector.IdempotencyContract{Strategy: connector.IdempotencyNatural}, Reconciliation: connector.ReconciliationNone, Compensation: connector.CompensationContract{Mode: connector.CompensationNone}},
}
var SendNotice = connector.EnqueueOperation[Request]{
	ConnectorKey: "member_center", ProviderKey: "acme",
	Key: "send_notice", ContractSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	Reliability: connector.ReliabilityContract{Effect: connector.EffectWrite, Idempotency: connector.IdempotencyContract{Strategy: connector.IdempotencyProviderKey, KeyRetentionSeconds: 86400}, Reconciliation: connector.ReconciliationProviderLookup, Compensation: connector.CompensationContract{Mode: connector.CompensationNone}},
}
var SyncMember = connector.StartOperation[Request]{
	ConnectorKey: "member_center", ProviderKey: "acme",
	Key: "sync_member", ContractSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	Reliability: connector.ReliabilityContract{Effect: connector.EffectWrite, Idempotency: connector.IdempotencyContract{Strategy: connector.IdempotencyProviderKey, KeyRetentionSeconds: 86400}, Reconciliation: connector.ReconciliationProviderLookup, Compensation: connector.CompensationContract{Mode: connector.CompensationNone}},
}

func Providers() (connector.ProviderSet, error) {
	call, err := connector.BindCall(GetMember, func(context.Context, connector.TypedRequest[Request]) (connector.TypedResult[Response], error) {
		return connector.TypedResult[Response]{Output: Response{Name: "Ada"}}, nil
	})
	if err != nil { return connector.ProviderSet{}, err }
	enqueue, err := connector.BindEnqueueDelivery(SendNotice, func(context.Context, connector.TypedRequest[Request]) (connector.DeliveryResult, error) {
		return connector.DeliveryResult{ResponseRef: "delivery-1"}, nil
	})
	if err != nil { return connector.ProviderSet{}, err }
	start, err := connector.BindStartOperationDelivery(SyncMember, func(context.Context, connector.TypedRequest[Request]) (connector.DeliveryResult, error) {
		return connector.DeliveryResult{ResponseRef: "remote-operation-1"}, nil
	})
	if err != nil { return connector.ProviderSet{}, err }
	provider, err := connector.NewProvider(connector.ProviderSchema{
		ConnectorKey: "member_center", ProviderKey: "acme", ProviderRevision: "provider-v1",
		ConfigFields: []connector.ConfigField{{Key: "region", Name: "Region", Type: connector.ConfigFieldSelect, Default: []byte(` + "`\"us\"`" + `), Validation: connector.ConfigValidation{Options: []string{"us", "eu"}}}},
		SecretFields: []connector.SecretField{{Key: "api_key", Name: "API key", Required: true, CredentialKind: connector.SecretCredentialAPIKey, MaterialFormat: connector.SecretMaterialOpaque, RotationPolicy: connector.SecretRotationManual, ExpiryPolicy: connector.SecretExpiryOptional, TestRequirement: connector.SecretTestWhenBound}},
	}, call, enqueue, start)
	if err != nil { return connector.ProviderSet{}, err }
	return connector.ProviderSet{Providers: []connector.Adapter{ProjectProvider{Adapter: provider}}}, nil
}
`)
	if err := os.WriteFile(filepath.Join(externalRoot, "connector.go"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = externalRoot
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("external project cannot compile connector: %v\n%s", err, output)
	}

	invalid := []byte(`package connector

import (
	"context"

	"github.com/domainry/domainry-connector-sdk"
)

func invalid(ctx context.Context, gateway connector.Gateway) {
	_, _ = connector.Call(ctx, gateway, SendNotice, Request{MemberID: "member-1"})
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

import "github.com/domainry/domainry-connector-sdk"

var _ connector.Invoker
`)
	if err := os.WriteFile(filepath.Join(externalRoot, "invalid.go"), noInvoker, 0o600); err != nil {
		t.Fatal(err)
	}
	command = exec.Command("go", "test", "./...")
	command.Dir = externalRoot
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err = command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "undefined: connector.Invoker") {
		t.Fatalf("public optional Invoker unexpectedly compiled: %v\n%s", err, output)
	}
}
