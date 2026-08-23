package connectorext

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev",
		ContractVersion: "connectorext-v17",
		ContractSHA256:  "ad163824c0115e814d754e6c537c0c7430e398f6c545b90d574ce498a80169de",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
