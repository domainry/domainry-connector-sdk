package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.1",
		ContractVersion: "connector-contract-v1",
		ContractSHA256:  "52e77dae1a2b09f210a2ee9597fe3bcbecb4ab875336e06a4a18ad3dfbcc90fb",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
