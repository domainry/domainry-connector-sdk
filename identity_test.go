package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.20",
		ContractVersion: "connector-contract-v20",
		ContractSHA256:  "2d8732a45c4ce7bd07413aeadab79dd1b39115846e8f2485f1feca4801427f8b",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
