package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.19",
		ContractVersion: "connector-contract-v17",
		ContractSHA256:  "18e3da970b4bf57d94601fd6f31fee1392b2c26b52a49e083b1b5d94f0fbf4db",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
