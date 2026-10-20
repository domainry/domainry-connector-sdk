package connector

import (
	"errors"
	"testing"

	"github.com/domainry/domainry-foundation/apperror"
)

func TestProviderErrorClassificationContract(t *testing.T) {
	tests := []struct {
		classification ErrorClassification
		kind           apperror.ErrorKind
		code           string
	}{
		{ErrorRetryable, apperror.KindUnavailable, "connector.provider_unavailable"},
		{ErrorUncertain, apperror.KindUnavailable, "connector.delivery_uncertain"},
		{ErrorPermanent, apperror.KindInternal, "connector.request_rejected"},
	}
	for _, test := range tests {
		err := NewProviderError(test.classification, test.code, errors.New("private cause"))
		if apperror.KindOf(err) != test.kind || apperror.CodeOf(err) != test.code {
			t.Fatalf("classification=%q kind=%q code=%q", test.classification, apperror.KindOf(err), apperror.CodeOf(err))
		}
	}
}
