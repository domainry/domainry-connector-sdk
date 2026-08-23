# Repository guide

- Do not use `domainry-builder-v1` to develop this repository.
- Keep this module independent from `domainry-plane` and `domainry-connectors`.
- Public contract changes require deterministic identity tests and external-module compile coverage.
- Do not start multiple local service instances. The SDK should not require a running service for tests.
