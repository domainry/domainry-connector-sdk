package connector

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
)

const ContractSHA256 = "52e77dae1a2b09f210a2ee9597fe3bcbecb4ab875336e06a4a18ad3dfbcc90fb"

const contractSurfaceV1 = `connector-contract-v1
PackagePath=github.com/domainry/domainry-connector-sdk
CallOperation[Input,Output]{ConnectorKey,ProviderKey,Key,ContractSHA256,Reliability}
EnqueueOperation[Input]{ConnectorKey,ProviderKey,Key,ContractSHA256,Reliability}
StartOperation[Input]{ConnectorKey,ProviderKey,Key,ContractSHA256,Reliability}
TypedRequest[Input]{Input,Connection,RequestRef,Headers,Secrets,Delivery,Principal}
TypedResult[Output]{Output,ResponseRef,SecretUpdates,ResourceHealth}
ResourceHealthReport=ExplicitProviderAPIOrStableProviderErrorOnly;AbsenceIsNeverInferred
BindCall(CallOperation[Input,Output],CallHandler[Input,Output])(BoundOperation,error)
BindEnqueueDelivery(EnqueueOperation[Input],DeliveryHandler[Input])(BoundOperation,error)
BindStartOperationDelivery(StartOperation[Input],DeliveryHandler[Input])(BoundOperation,error)
HandlerErrorResult=PreserveResponseRefAndDescriptorScopedSecretUpdatesWithoutPayload
ProviderHealthOnSuccessOrError=PreserveExplicitResourceHealthWithoutInference
Call(context.Context,Gateway,CallOperation[Input,Output],Input)(Output,error)
Gateway.Call(context.Context,CallRequest)(CallResult,error)
Adapter.Descriptor()ProviderDescriptor
Adapter.Call(context.Context,CallRequest)(CallResult,error)
ConfigValidator.ValidateConfig(Connection)error
ConnectionTester.TestConnection(context.Context,TestConnectionRequest)(TestConnectionResult,error)
WebhookVerifier.VerifyWebhook(context.Context,VerifyWebhookRequest)(VerifiedWebhook,error)
NoOptionalInvoker=Adapter.CallIsTheOnlyProviderOperationExecutionEntry
ErrorClassification=retryable,permanent,uncertain
NewProviderError(ErrorClassification,string,error)error
RetryableError(string,error)error
PermanentError(string,error)error
UncertainError(string,error)error
ErrorClassificationOf(error)(ErrorClassification,bool)
ProviderErrorCodeOf(error)(string,bool)
ProviderError.Error()=StableCodeOnly
ProviderError.Unwrap()=OriginalCause
UnclassifiedErrors=NeverInferredFromMessage
Retryable=EligibleOnlyAfterRuntimeIdempotencyPolicy
Uncertain=NeverBlindRetryRequiresReconciliation
ReliabilityContract={Effect,Idempotency,Reconciliation,Compensation}
ReliabilityContract.Validate(string)error
OperationEffect=read,reserve,write
IdempotencyStrategy=none,natural,provider_key
ProviderKeyIdempotency=PositiveKeyRetentionSeconds
ReconciliationMode=none,provider_lookup
ReconciliationOutcome=succeeded,failed,pending,not_found,unknown
Reconciler.Reconcile(context.Context,ReconcileRequest)(ReconcileResult,error)
ReconcileRequest.Validate()error
ReconcileResult.Validate()error
CompensationMode=none,explicit,saga
CompensationTarget=DistinctTerminalIdempotentProviderReconcilableEnqueueWriteOperation
CompensationExecution=Adapter.CallViaCommittedOutbox
ProviderSchema.Validate()error
StartupActivationPolicy=manual,default_safe
DefaultSafeStartup=ExplicitProviderSchemaOnly,NoRequiredSecrets,AllRequiredConfigDefaulted
ProviderDescriptor.RequiresReconciler()bool
ConfigField{Key,Name,Description,Type,I18n,Required,Default,Validation,RequiredWith}
SecretField{Key,Name,Description,I18n,Required,CredentialKind,MaterialFormat,RotationPolicy,ExpiryPolicy,TestRequirement}
ConfigFieldType=text,email,integer,decimal,boolean,select,json
SecretCredentialKind=api_key,basic_auth_password,bearer_token,certificate,connection_string,database_password,generic_secret,identifier,oauth_client_secret,private_key,refresh_token,service_account,signing_secret
SecretMaterialFormat=opaque,text,json_object,pem,pem_or_opaque,pem_or_reference,uri_or_dsn
SecretRotationPolicy=manual,oauth_refresh
SecretExpiryPolicy=none,optional,required
SecretTestRequirement=optional,when_bound
NewProvider(ProviderSchema,...BoundOperation)(Adapter,error)
Registry.Register(Adapter)error
Registry.RegisterExtensionSet(ExtensionSet)error
RegistryRegistration=ValidatedDescriptorSnapshotAndExactOptionalCapabilitiesIncludingConfigValidator,ConnectionTester,WebhookVerifier,Reconciler
Registry.Freeze()
Registry.Provider(string,string)(Adapter,bool)
Registry.Descriptors()[]ProviderDescriptor
Registry.Providers()[]Adapter
Transport.RoundTripHTTP(context.Context,HTTPRequest)(HTTPResponse,error)
Transport.ExecuteSQL(context.Context,SQLRequest)(SQLResult,error)
SQLOperation=ping,query,exec
ExtensionFactory(Transport)(ExtensionSet,error)
ProjectAdapterConstructor=New*Adapter(Transport)Adapter
RuntimeSecretResolution=AllReferencesResolvedBeforeAdapterInvocation
AdapterSecretMaterial=ProviderDescriptorSecretFieldsOnly
AdapterConnectionSecretRefs=ProviderDescriptorSecretFieldsOnly
AdapterSecretUpdates=ProviderDescriptorSecretFieldsOnly
`

func ComputedContractSHA256() string {
	structs := []any{
		Connection{}, Principal{}, CallRequest{}, CallResult{}, ResourceHealthReport{}, TestConnectionRequest{}, TestConnectionResult{}, VerifyWebhookRequest{}, WebhookSecurityEvidence{}, WebhookExternalIdentity{}, WebhookDeliveryReceipt{}, VerifiedWebhook{}, ProviderError{}, IdempotencyContract{}, CompensationContract{}, ReliabilityContract{}, ReconcileRequest{}, ReconcileResult{},
		DeliveryResult{}, OperationDescriptor{}, FieldLocalization{}, ConfigValidation{}, ConfigField{}, SecretField{}, ProviderSchema{}, ProviderDescriptor{}, ExtensionSet{},
		HTTPRequest{}, HTTPResponse{}, SQLRequest{}, SQLResult{},
	}
	var surface strings.Builder
	surface.WriteString(contractSurfaceV1)
	for _, value := range structs {
		current := reflect.TypeOf(value)
		surface.WriteString(current.Name())
		surface.WriteByte('{')
		for index := 0; index < current.NumField(); index++ {
			if index > 0 {
				surface.WriteByte(',')
			}
			field := current.Field(index)
			surface.WriteString(field.Name)
			surface.WriteByte(':')
			surface.WriteString(field.Type.String())
			surface.WriteByte(':')
			surface.WriteString(field.Tag.Get("json"))
		}
		surface.WriteString("}\n")
	}
	sum := sha256.Sum256([]byte(surface.String()))
	return hex.EncodeToString(sum[:])
}
