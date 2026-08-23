package minimal

import (
	"testing"

	"github.com/domainry/domainry-connector-sdk/contracttest"
)

func TestProvidersSatisfyContract(t *testing.T) {
	providers, err := Providers(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(providers.Providers))
	}
	if err := contracttest.ValidateAdapter(providers.Providers[0]); err != nil {
		t.Fatal(err)
	}
}
