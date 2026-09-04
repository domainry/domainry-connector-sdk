package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.14",
		ContractVersion: "connector-contract-v13",
		ContractSHA256:  "74f94a917680e0d87a05b9a9c34a3142405270b5ab0ac4effb2067042b41ba66",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
