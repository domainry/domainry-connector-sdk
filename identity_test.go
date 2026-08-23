package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.2",
		ContractVersion: "connector-contract-v2",
		ContractSHA256:  "7c0840aba1388add0ec482254f144349e1ee309b5480f368f9d32a0218903d1d",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
