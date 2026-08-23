package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.5",
		ContractVersion: "connector-contract-v5",
		ContractSHA256:  "ed0e5dc7db5c8d89b3d6d8f7689ade7fde1afb0ad7722431c275430be7bc6499",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
