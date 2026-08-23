package connectorext

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type ConfigFieldType string

const (
	ConfigFieldText    ConfigFieldType = "text"
	ConfigFieldEmail   ConfigFieldType = "email"
	ConfigFieldInteger ConfigFieldType = "integer"
	ConfigFieldDecimal ConfigFieldType = "decimal"
	ConfigFieldBoolean ConfigFieldType = "boolean"
	ConfigFieldSelect  ConfigFieldType = "select"
	ConfigFieldJSON    ConfigFieldType = "json"
)

type FieldLocalization struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type ConfigValidation struct {
	MinLength int      `json:"min_length,omitempty"`
	MaxLength int      `json:"max_length,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Options   []string `json:"options,omitempty"`
}

type ConfigField struct {
	Key          string                       `json:"key"`
	Name         string                       `json:"name"`
	Description  string                       `json:"description,omitempty"`
	Type         ConfigFieldType              `json:"type"`
	I18n         map[string]FieldLocalization `json:"i18n,omitempty"`
	Required     bool                         `json:"required"`
	Default      json.RawMessage              `json:"default,omitempty"`
	Validation   ConfigValidation             `json:"validation,omitempty"`
	RequiredWith []string                     `json:"required_with,omitempty"`
}

type SecretCredentialKind string

const (
	SecretCredentialAPIKey            SecretCredentialKind = "api_key"
	SecretCredentialBasicAuthPassword SecretCredentialKind = "basic_auth_password"
	SecretCredentialBearerToken       SecretCredentialKind = "bearer_token"
	SecretCredentialCertificate       SecretCredentialKind = "certificate"
	SecretCredentialConnectionString  SecretCredentialKind = "connection_string"
	SecretCredentialDatabasePassword  SecretCredentialKind = "database_password"
	SecretCredentialGeneric           SecretCredentialKind = "generic_secret"
	SecretCredentialIdentifier        SecretCredentialKind = "identifier"
	SecretCredentialOAuthClientSecret SecretCredentialKind = "oauth_client_secret"
	SecretCredentialPrivateKey        SecretCredentialKind = "private_key"
	SecretCredentialRefreshToken      SecretCredentialKind = "refresh_token"
	SecretCredentialServiceAccount    SecretCredentialKind = "service_account"
	SecretCredentialSigningSecret     SecretCredentialKind = "signing_secret"
)

type SecretMaterialFormat string

const (
	SecretMaterialOpaque         SecretMaterialFormat = "opaque"
	SecretMaterialText           SecretMaterialFormat = "text"
	SecretMaterialJSONObject     SecretMaterialFormat = "json_object"
	SecretMaterialPEM            SecretMaterialFormat = "pem"
	SecretMaterialPEMOrOpaque    SecretMaterialFormat = "pem_or_opaque"
	SecretMaterialPEMOrReference SecretMaterialFormat = "pem_or_reference"
	SecretMaterialURIOrDSN       SecretMaterialFormat = "uri_or_dsn"
)

type SecretRotationPolicy string

const (
	SecretRotationManual       SecretRotationPolicy = "manual"
	SecretRotationOAuthRefresh SecretRotationPolicy = "oauth_refresh"
)

type SecretExpiryPolicy string

const (
	SecretExpiryNone     SecretExpiryPolicy = "none"
	SecretExpiryOptional SecretExpiryPolicy = "optional"
	SecretExpiryRequired SecretExpiryPolicy = "required"
)

type SecretTestRequirement string

const (
	SecretTestOptional  SecretTestRequirement = "optional"
	SecretTestWhenBound SecretTestRequirement = "when_bound"
)

type SecretField struct {
	Key             string                       `json:"key"`
	Name            string                       `json:"name"`
	Description     string                       `json:"description,omitempty"`
	I18n            map[string]FieldLocalization `json:"i18n,omitempty"`
	Required        bool                         `json:"required"`
	CredentialKind  SecretCredentialKind         `json:"credential_kind"`
	MaterialFormat  SecretMaterialFormat         `json:"material_format"`
	RotationPolicy  SecretRotationPolicy         `json:"rotation_policy"`
	ExpiryPolicy    SecretExpiryPolicy           `json:"expiry_policy"`
	TestRequirement SecretTestRequirement        `json:"test_requirement"`
}

type ProviderSchema struct {
	ConnectorKey      string                  `json:"connector_key"`
	ProviderKey       string                  `json:"provider_key"`
	ProviderRevision  string                  `json:"provider_revision"`
	StartupActivation StartupActivationPolicy `json:"startup_activation,omitempty"`
	ConfigFields      []ConfigField           `json:"config_fields,omitempty"`
	SecretFields      []SecretField           `json:"secret_fields,omitempty"`
}

type StartupActivationPolicy string

const (
	StartupActivationManual      StartupActivationPolicy = ""
	StartupActivationDefaultSafe StartupActivationPolicy = "default_safe"
)

func (s ProviderSchema) Validate() error {
	if !publicIdentityKey(s.ConnectorKey) || !publicIdentityKey(s.ProviderKey) {
		return errors.New("connector and provider keys must be canonical lowercase identities")
	}
	if strings.TrimSpace(s.ProviderRevision) == "" || s.ProviderRevision != strings.TrimSpace(s.ProviderRevision) {
		return fmt.Errorf("connector provider %s/%s revision is required", s.ConnectorKey, s.ProviderKey)
	}
	if s.StartupActivation != StartupActivationManual && s.StartupActivation != StartupActivationDefaultSafe {
		return fmt.Errorf("connector provider %s/%s has invalid startup activation %q", s.ConnectorKey, s.ProviderKey, s.StartupActivation)
	}
	configKeys := make(map[string]bool, len(s.ConfigFields))
	for _, field := range s.ConfigFields {
		if err := field.validate(); err != nil {
			return fmt.Errorf("connector provider %s/%s config field: %w", s.ConnectorKey, s.ProviderKey, err)
		}
		if configKeys[field.Key] {
			return fmt.Errorf("connector provider %s/%s config field %s is duplicated", s.ConnectorKey, s.ProviderKey, field.Key)
		}
		configKeys[field.Key] = true
	}
	secretKeys := make(map[string]bool, len(s.SecretFields))
	for _, field := range s.SecretFields {
		if err := field.validate(); err != nil {
			return fmt.Errorf("connector provider %s/%s secret field: %w", s.ConnectorKey, s.ProviderKey, err)
		}
		if configKeys[field.Key] || secretKeys[field.Key] {
			return fmt.Errorf("connector provider %s/%s field %s is duplicated", s.ConnectorKey, s.ProviderKey, field.Key)
		}
		secretKeys[field.Key] = true
	}
	for _, field := range s.ConfigFields {
		for _, dependency := range field.RequiredWith {
			if !configKeys[dependency] {
				return fmt.Errorf("connector provider %s/%s config field %s requires unknown config field %s", s.ConnectorKey, s.ProviderKey, field.Key, dependency)
			}
		}
	}
	if s.StartupActivation == StartupActivationDefaultSafe {
		for _, field := range s.ConfigFields {
			if field.Required && len(field.Default) == 0 {
				return fmt.Errorf("connector provider %s/%s default-safe startup requires a default for required config field %s", s.ConnectorKey, s.ProviderKey, field.Key)
			}
		}
		for _, field := range s.SecretFields {
			if field.Required {
				return fmt.Errorf("connector provider %s/%s default-safe startup cannot require secret field %s", s.ConnectorKey, s.ProviderKey, field.Key)
			}
		}
	}
	return nil
}

func (f ConfigField) validate() error {
	if !publicIdentityKey(f.Key) || strings.TrimSpace(f.Name) == "" {
		return fmt.Errorf("field %q requires a canonical key and name", f.Key)
	}
	if !validConfigFieldType(f.Type) {
		return fmt.Errorf("field %s has invalid type %q", f.Key, f.Type)
	}
	if err := validateFieldLocalization(f.Key, f.I18n); err != nil {
		return err
	}
	if err := validateConfigRules(f); err != nil {
		return err
	}
	if len(f.Default) > 0 {
		value, err := decodeConfigDefault(f.Default)
		if err != nil {
			return fmt.Errorf("field %s default is invalid: %w", f.Key, err)
		}
		if !configDefaultMatchesType(value, f.Type) {
			return fmt.Errorf("field %s default does not match type %s", f.Key, f.Type)
		}
		if err := validateConfigDefaultRules(f, value); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	for _, dependency := range f.RequiredWith {
		if !publicIdentityKey(dependency) || dependency == f.Key || seen[dependency] {
			return fmt.Errorf("field %s has invalid required_with field %q", f.Key, dependency)
		}
		seen[dependency] = true
	}
	return nil
}

func (f SecretField) validate() error {
	if !publicIdentityKey(f.Key) || strings.TrimSpace(f.Name) == "" {
		return fmt.Errorf("field %q requires a canonical key and name", f.Key)
	}
	if err := validateFieldLocalization(f.Key, f.I18n); err != nil {
		return err
	}
	if !validSecretCredentialKind(f.CredentialKind) || !validSecretMaterialFormat(f.MaterialFormat) || !validSecretRotationPolicy(f.RotationPolicy) || !validSecretExpiryPolicy(f.ExpiryPolicy) || !validSecretTestRequirement(f.TestRequirement) {
		return fmt.Errorf("field %s has invalid secret lifecycle metadata", f.Key)
	}
	return nil
}

func validateConfigRules(field ConfigField) error {
	rules := field.Validation
	textRules := rules.MinLength != 0 || rules.MaxLength != 0 || strings.TrimSpace(rules.Pattern) != ""
	numberRules := rules.Min != nil || rules.Max != nil
	if textRules && field.Type != ConfigFieldText && field.Type != ConfigFieldEmail {
		return fmt.Errorf("field %s text validation requires text or email type", field.Key)
	}
	if numberRules && field.Type != ConfigFieldInteger && field.Type != ConfigFieldDecimal {
		return fmt.Errorf("field %s numeric validation requires integer or decimal type", field.Key)
	}
	if rules.MinLength < 0 || rules.MaxLength < 0 || (rules.MaxLength > 0 && rules.MinLength > rules.MaxLength) {
		return fmt.Errorf("field %s has invalid length bounds", field.Key)
	}
	if (rules.Min != nil && (math.IsNaN(*rules.Min) || math.IsInf(*rules.Min, 0))) || (rules.Max != nil && (math.IsNaN(*rules.Max) || math.IsInf(*rules.Max, 0))) {
		return fmt.Errorf("field %s has non-finite numeric bounds", field.Key)
	}
	if rules.Min != nil && rules.Max != nil && *rules.Min > *rules.Max {
		return fmt.Errorf("field %s has invalid numeric bounds", field.Key)
	}
	if strings.TrimSpace(rules.Pattern) != "" {
		if _, err := regexp.Compile(rules.Pattern); err != nil {
			return fmt.Errorf("field %s has invalid pattern", field.Key)
		}
	}
	seen := map[string]bool{}
	for _, option := range rules.Options {
		if strings.TrimSpace(option) == "" || option != strings.TrimSpace(option) || seen[option] {
			return fmt.Errorf("field %s has invalid option %q", field.Key, option)
		}
		seen[option] = true
	}
	if field.Type == ConfigFieldSelect && len(rules.Options) == 0 {
		return fmt.Errorf("field %s select type requires options", field.Key)
	}
	if field.Type != ConfigFieldSelect && len(rules.Options) > 0 {
		return fmt.Errorf("field %s options require select type", field.Key)
	}
	return nil
}

func validateConfigDefaultRules(field ConfigField, value any) error {
	rules := field.Validation
	if field.Type == ConfigFieldSelect && !containsString(rules.Options, value.(string)) {
		return fmt.Errorf("field %s default is not one of its options", field.Key)
	}
	if text, ok := value.(string); ok {
		length := len([]rune(text))
		if (rules.MinLength > 0 && length < rules.MinLength) || (rules.MaxLength > 0 && length > rules.MaxLength) {
			return fmt.Errorf("field %s default violates length bounds", field.Key)
		}
		if strings.TrimSpace(rules.Pattern) != "" && !regexp.MustCompile(rules.Pattern).MatchString(text) {
			return fmt.Errorf("field %s default violates pattern", field.Key)
		}
	}
	if number, ok := value.(json.Number); ok {
		parsed, err := strconv.ParseFloat(string(number), 64)
		if err != nil || (rules.Min != nil && parsed < *rules.Min) || (rules.Max != nil && parsed > *rules.Max) {
			return fmt.Errorf("field %s default violates numeric bounds", field.Key)
		}
	}
	return nil
}

func decodeConfigDefault(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("null is not a usable default")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("default must contain exactly one JSON value")
	}
	return value, nil
}

// ApplyConfigDefaults materializes schema-owned, non-secret defaults without
// replacing any value already supplied by a manifest, tenant, or administrator.
// Decoding on every call also gives mutable JSON defaults an independent value.
func ApplyConfigDefaults(fields []ConfigField, config map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(config)+len(fields))
	for key, value := range config {
		result[key] = value
	}
	for _, field := range fields {
		if _, exists := result[field.Key]; exists || len(field.Default) == 0 {
			continue
		}
		ready := true
		for _, dependency := range field.RequiredWith {
			if !configValuePresent(result, dependency) {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		value, err := decodeConfigDefault(field.Default)
		if err != nil {
			return nil, fmt.Errorf("config field %s default is invalid: %w", field.Key, err)
		}
		result[field.Key] = value
	}
	return result, nil
}

func configValuePresent(config map[string]any, key string) bool {
	value, exists := config[strings.TrimSpace(key)]
	if !exists || value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func configDefaultMatchesType(value any, fieldType ConfigFieldType) bool {
	switch fieldType {
	case ConfigFieldText, ConfigFieldEmail, ConfigFieldSelect:
		_, ok := value.(string)
		return ok
	case ConfigFieldInteger:
		number, ok := value.(json.Number)
		if !ok {
			return false
		}
		_, err := strconv.ParseInt(string(number), 10, 64)
		return err == nil
	case ConfigFieldDecimal:
		number, ok := value.(json.Number)
		if !ok {
			return false
		}
		parsed, err := strconv.ParseFloat(string(number), 64)
		return err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
	case ConfigFieldBoolean:
		_, ok := value.(bool)
		return ok
	case ConfigFieldJSON:
		return true
	default:
		return false
	}
}

func validateFieldLocalization(key string, localized map[string]FieldLocalization) error {
	for locale, value := range localized {
		if strings.TrimSpace(locale) == "" || locale != strings.TrimSpace(locale) || strings.TrimSpace(value.Name) == "" && strings.TrimSpace(value.Description) == "" {
			return fmt.Errorf("field %s has invalid localization %q", key, locale)
		}
	}
	return nil
}

func validConfigFieldType(value ConfigFieldType) bool {
	switch value {
	case ConfigFieldText, ConfigFieldEmail, ConfigFieldInteger, ConfigFieldDecimal, ConfigFieldBoolean, ConfigFieldSelect, ConfigFieldJSON:
		return true
	default:
		return false
	}
}

func validSecretCredentialKind(value SecretCredentialKind) bool {
	switch value {
	case SecretCredentialAPIKey, SecretCredentialBasicAuthPassword, SecretCredentialBearerToken, SecretCredentialCertificate, SecretCredentialConnectionString, SecretCredentialDatabasePassword, SecretCredentialGeneric, SecretCredentialIdentifier, SecretCredentialOAuthClientSecret, SecretCredentialPrivateKey, SecretCredentialRefreshToken, SecretCredentialServiceAccount, SecretCredentialSigningSecret:
		return true
	default:
		return false
	}
}

func validSecretMaterialFormat(value SecretMaterialFormat) bool {
	switch value {
	case SecretMaterialOpaque, SecretMaterialText, SecretMaterialJSONObject, SecretMaterialPEM, SecretMaterialPEMOrOpaque, SecretMaterialPEMOrReference, SecretMaterialURIOrDSN:
		return true
	default:
		return false
	}
}

func validSecretRotationPolicy(value SecretRotationPolicy) bool {
	return value == SecretRotationManual || value == SecretRotationOAuthRefresh
}

func validSecretExpiryPolicy(value SecretExpiryPolicy) bool {
	return value == SecretExpiryNone || value == SecretExpiryOptional || value == SecretExpiryRequired
}

func validSecretTestRequirement(value SecretTestRequirement) bool {
	return value == SecretTestOptional || value == SecretTestWhenBound
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var publicIdentityPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,127}$`)

func publicIdentityKey(value string) bool {
	return publicIdentityPattern.MatchString(value)
}
