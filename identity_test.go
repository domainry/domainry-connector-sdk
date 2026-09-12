package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.19",
		ContractVersion: "connector-contract-v16",
		ContractSHA256:  "664cb625e96fed3fcfd88c6b25e0087c63aa18e22dd2c960ab9d858b85941fee",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
