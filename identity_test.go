package connector

import "testing"

func TestCurrentIdentity(t *testing.T) {
	want := Identity{
		SDKVersion:      "v0.1.0-dev.5",
		ContractVersion: "connector-contract-v5",
		ContractSHA256:  "916f09dda181c978735eadf1df35cef251a07fe4556b033e207181ba9ce319da",
	}
	if got := CurrentIdentity(); got != want {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, want)
	}
}
