package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.3",
		ContractVersion: "connector-contract-v3",
		ContractSHA256:  "129494b793676175f1d67814fa8be1507034e868ec6b733c36b08020f5a2771e",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
