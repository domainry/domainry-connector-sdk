package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.10",
		ContractVersion: "connector-contract-v9",
		ContractSHA256:  "8d3662131e497098735d876e81b96cf4ac9146acd15b74a92fcc9bcfd2ae7504",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
