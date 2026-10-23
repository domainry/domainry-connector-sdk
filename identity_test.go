package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.21",
		ContractVersion: "connector-contract-v21",
		ContractSHA256:  "5f6a6cba1f3def1d850219fa482130e70d04a7836ee2b5412656d4a2fba9480e",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
