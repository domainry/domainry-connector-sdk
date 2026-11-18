package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.23",
		ContractVersion: "connector-contract-v23",
		ContractSHA256:  "bcec5c0ae3c2cf18d3819144ba41365ae2fc32859895de4e2407af3bdd4fffe4",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
