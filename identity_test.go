package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.12",
		ContractVersion: "connector-contract-v10",
		ContractSHA256:  "46b776e062af85c4b6d39f6d8bfaab08800454a6ab52a1077aaa3eda87f2fdf6",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
