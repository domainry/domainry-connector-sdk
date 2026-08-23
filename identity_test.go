package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.7",
		ContractVersion: "connector-contract-v7",
		ContractSHA256:  "7f2f6bb06bab4143faf75ebffa76f44cc1f06de6ecdbcfeb88ca6604a5b35274",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
