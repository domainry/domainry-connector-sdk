package connector

// SDKVersion identifies the release line of this module independently from
// the compatibility identity of its public Connector contract.
const SDKVersion = "v0.1.0-dev.10"

// Identity is recorded by hosts and generated projects so semantic SDK
// releases cannot accidentally hide a public contract change.
type Identity struct {
	SDKVersion      string `json:"sdk_version"`
	ContractVersion string `json:"contract_version"`
	ContractSHA256  string `json:"contract_sha256"`
}

// CurrentIdentity returns the immutable identity of this SDK build.
func CurrentIdentity() Identity {
	return Identity{
		SDKVersion:      SDKVersion,
		ContractVersion: ContractVersion,
		ContractSHA256:  ContractSHA256,
	}
}
