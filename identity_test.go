package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.22",
		ContractVersion: "connector-contract-v22",
		ContractSHA256:  "88741d653e4d03182735ee5e1d6ae0ab0b0d718a630d4a9eadf4d9c63a702ddf",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
