# Domainry Connector SDK for Go

This repository owns the stable public contracts used by Domainry Plane,
official Domainry Connectors, reviewed third-party Connectors, and
customer-private Connectors.

The SDK defines Provider descriptors, operations, reliability semantics,
host-supplied transports, Provider factories, webhook verification,
connection testing, reconciliation, error classification, and contract-test
helpers. It does not implement Integration persistence, authorization, audit,
secret storage, scheduling, Admin surfaces, Builder behavior, or concrete
Providers. Connector definitions and implementations live in
`domainry-connectors`; `domainry-integration` owns their configured runtime
state and execution.

## Dependency boundary

The SDK must not import either of these modules:

- `github.com/domainry/domainry-plane`
- `github.com/domainry/domainry-connectors`

Both modules depend on this SDK instead.

## Package layout

- The root package owns provider, gateway, adapter, transport, reliability, and background-processing contracts.
- `contracttest` contains reusable implementation conformance tests.
- `examples` contains public integration examples.

The SDK intentionally has no `persistence` or `modulehost` package: the host
supplies governed transport capabilities, Integration owns connection and
execution infrastructure, and concrete Connectors own Provider implementations.

## Development

```sh
make fmt-check
make test
make vet
make boundary
make license-check
make vulnerability-check
```

`make release-check` runs every required release gate. Public compatibility is
identified by `SDKVersion`, `ContractVersion`, and `ContractSHA256`; consumers
should depend on an immutable tagged version.

## License

The SDK is available under the MIT License. See [LICENSE](LICENSE).
