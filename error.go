package connector

import (
	"errors"
	"fmt"
	"regexp"
)

// ErrorClassification is the closed recovery decision reported by provider
// code. Runtime still applies the operation's idempotency and retry policy; a
// retryable classification alone never authorizes a retry.
type ErrorClassification string

const (
	ErrorRetryable ErrorClassification = "retryable"
	ErrorPermanent ErrorClassification = "permanent"
	ErrorUncertain ErrorClassification = "uncertain"
)

const ErrorContractInvalidCode = "connector.error_contract_invalid"

var providerErrorCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)

// ProviderError carries machine-readable recovery semantics. Error deliberately
// returns only Code: provider response text and wrapped causes may contain
// credentials or customer data and are not a public/logging contract.
type ProviderError struct {
	classification ErrorClassification
	code           string
	cause          error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *ProviderError) Classification() ErrorClassification {
	if e == nil {
		return ""
	}
	return e.classification
}

func (e *ProviderError) Code() string {
	if e == nil {
		return ""
	}
	return e.code
}

// NewProviderError creates a classified provider failure. Invalid class/code
// input is converted to a permanent contract error rather than guessed or
// allowed to become retryable.
func NewProviderError(classification ErrorClassification, code string, cause error) error {
	if err := validateProviderError(classification, code); err != nil {
		if cause != nil {
			err = fmt.Errorf("%w: %v", cause, err)
		}
		return &ProviderError{classification: ErrorPermanent, code: ErrorContractInvalidCode, cause: err}
	}
	return &ProviderError{classification: classification, code: code, cause: cause}
}

func RetryableError(code string, cause error) error {
	return NewProviderError(ErrorRetryable, code, cause)
}

func PermanentError(code string, cause error) error {
	return NewProviderError(ErrorPermanent, code, cause)
}

func UncertainError(code string, cause error) error {
	return NewProviderError(ErrorUncertain, code, cause)
}

func ErrorClassificationOf(err error) (ErrorClassification, bool) {
	var providerError *ProviderError
	if !errors.As(err, &providerError) || providerError == nil {
		return "", false
	}
	if validateProviderError(providerError.classification, providerError.code) != nil {
		return "", false
	}
	return providerError.classification, true
}

func ProviderErrorCodeOf(err error) (string, bool) {
	var providerError *ProviderError
	if !errors.As(err, &providerError) || providerError == nil {
		return "", false
	}
	if validateProviderError(providerError.classification, providerError.code) != nil {
		return "", false
	}
	return providerError.code, true
}

func validateProviderError(classification ErrorClassification, code string) error {
	switch classification {
	case ErrorRetryable, ErrorPermanent, ErrorUncertain:
	default:
		return fmt.Errorf("connector provider error classification %q is invalid", classification)
	}
	if !providerErrorCodePattern.MatchString(code) {
		return fmt.Errorf("connector provider error code %q must be a namespaced lower_snake_case token", code)
	}
	return nil
}
