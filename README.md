# Domainry Connector SDK for Go

This repository owns the stable public contracts used by Domainry Plane,
official Domainry Connectors, reviewed third-party Connectors, and
customer-private Connectors.

The SDK defines Provider descriptors, operations, reliability semantics,
Runtime-owned transports, extension factories, webhook verification,
connection testing, reconciliation, error classification, and contract-test
helpers. It does not implement Runtime persistence, authorization, audit,
secret storage, scheduling, Admin surfaces, Builder behavior, or concrete
Providers.

## Dependency boundary

The SDK must not import either of these modules:

- `github.com/domainry/domainry-plane`
- `github.com/domainry/domainry-connectors`

Both modules depend on this SDK instead.

## Development

```sh
make fmt-check
make test
make vet
make boundary
```

The module is currently under initial extraction. Public compatibility begins
only when the first tagged SDK contract is released.

## License

No public license is granted during the extraction phase. A `LICENSE` file
must be added through an explicit repository policy decision before the first
public release.
