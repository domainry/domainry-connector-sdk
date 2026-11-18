package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.24",
		ContractVersion: "connector-contract-v24",
		ContractSHA256:  "190b7a570ae747ea04ff16d77a8cde127b6cc5215dd68f4d951bd70bef40075e",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
