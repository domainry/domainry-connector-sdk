package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.19",
		ContractVersion: "connector-contract-v18",
		ContractSHA256:  "fee4768c9d7216ebbb4453f987f8e360805843c9c11eba1602c7df5dd5332a9f",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
