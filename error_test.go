package connector

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestProviderErrorHasClosedExplicitRecoveryClassification(t *testing.T) {
	cause := context.DeadlineExceeded
	tests := []struct {
		name           string
		err            error
		classification ErrorClassification
		code           string
	}{
		{name: "retryable", err: RetryableError("acme.rate_limited", cause), classification: ErrorRetryable, code: "acme.rate_limited"},
		{name: "permanent", err: PermanentError("acme.request_rejected", cause), classification: ErrorPermanent, code: "acme.request_rejected"},
		{name: "uncertain", err: UncertainError("acme.delivery_unknown", cause), classification: ErrorUncertain, code: "acme.delivery_unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.err.Error() != test.code {
				t.Fatalf("Error leaked cause or changed code: %q", test.err.Error())
			}
			if !errors.Is(test.err, cause) {
				t.Fatalf("cause chain was not preserved: %v", test.err)
			}
			classification, classified := ErrorClassificationOf(fmt.Errorf("provider wrapper: %w", test.err))
			code, coded := ProviderErrorCodeOf(fmt.Errorf("provider wrapper: %w", test.err))
			if !classified || classification != test.classification || !coded || code != test.code {
				t.Fatalf("classification=%q/%v code=%q/%v", classification, classified, code, coded)
			}
			var providerError *ProviderError
			if !errors.As(test.err, &providerError) || providerError.Classification() != test.classification || providerError.Code() != test.code {
				t.Fatalf("typed provider error=%+v", providerError)
			}
		})
	}
}

func TestProviderErrorInvalidContractFailsClosedAsPermanent(t *testing.T) {
	cause := errors.New("secret provider response")
	tests := []error{
		NewProviderError(ErrorClassification("transient"), "acme.timeout", cause),
		NewProviderError(ErrorRetryable, "", cause),
		NewProviderError(ErrorRetryable, "Human readable provider failure", cause),
		NewProviderError(ErrorRetryable, "ACME.Timeout", cause),
		NewProviderError(ErrorRetryable, "timeout", cause),
		NewProviderError(ErrorRetryable, " acme.timeout ", cause),
	}
	for _, err := range tests {
		if err.Error() != ErrorContractInvalidCode || !errors.Is(err, cause) {
			t.Fatalf("invalid contract error=%q cause=%v", err, errors.Is(err, cause))
		}
		classification, classified := ErrorClassificationOf(err)
		code, coded := ProviderErrorCodeOf(err)
		if !classified || classification != ErrorPermanent || !coded || code != ErrorContractInvalidCode {
			t.Fatalf("invalid contract classification=%q/%v code=%q/%v", classification, classified, code, coded)
		}
	}
}

func TestUnknownErrorsAreNotClassifiedFromText(t *testing.T) {
	for _, err := range []error{
		nil,
		errors.New("timeout"),
		errors.New("rate_limited"),
		errors.New("connection reset after provider success"),
		context.DeadlineExceeded,
	} {
		if classification, ok := ErrorClassificationOf(err); ok || classification != "" {
			t.Fatalf("unknown error %v was classified as %q", err, classification)
		}
		if code, ok := ProviderErrorCodeOf(err); ok || code != "" {
			t.Fatalf("unknown error %v exposed code %q", err, code)
		}
	}
}

func TestNilProviderErrorAccessorsAreSafe(t *testing.T) {
	var err *ProviderError
	if err.Error() != "" || err.Unwrap() != nil || err.Classification() != "" || err.Code() != "" {
		t.Fatal("nil ProviderError accessors are not zero-safe")
	}
}
