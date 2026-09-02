package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.14",
		ContractVersion: "connector-contract-v12",
		ContractSHA256:  "9b4ab932cb52396a4901938c70bda77feb3564e8686b6b70e72e973e169964e3",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
