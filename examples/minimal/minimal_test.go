package minimal

import (
	"testing"

	"github.com/domainry/domainry-connector-sdk/contracttest"
)

func TestExtensionsSatisfyContract(t *testing.T) {
	extensions, err := Extensions(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(extensions.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(extensions.Providers))
	}
	if err := contracttest.ValidateAdapter(extensions.Providers[0]); err != nil {
		t.Fatal(err)
	}
}
