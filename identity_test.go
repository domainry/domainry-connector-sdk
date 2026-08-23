package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.4",
		ContractVersion: "connector-contract-v4",
		ContractSHA256:  "e601024f36d4d863a73abc792a2f7a7b488b91ddf2c032e53fbd0702df6373ea",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
