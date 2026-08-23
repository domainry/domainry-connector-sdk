package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.8",
		ContractVersion: "connector-contract-v8",
		ContractSHA256:  "c0934e9e6d5ce24f8be0cfe1df1a3bd32bbd818924bf818706943792c04fe33e",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
