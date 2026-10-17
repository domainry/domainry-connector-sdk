package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.19",
		ContractVersion: "connector-contract-v19",
		ContractSHA256:  "176f60ef8281c9c261ebb25d4bda7f08551cdc9b43ce7ffa41e2afd30046b6b5",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
