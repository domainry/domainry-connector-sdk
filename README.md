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

## OAuth connection-test requirements

Providers may implement `OAuthConnectionTestScopeProvider` for their fixed
connection probe. Each outer element is an alternative; every scope within an
alternative is required. A declared empty alternative means the probe needs no
OAuth scopes. An absent declaration retains legacy behavior. The registry copies
both slice levels at registration and on reads. Integration evaluates these
requirements against the actual grant; this metadata never requests extra scopes
or decides access to business tools. The optional authorizer interface is unchanged.

## OAuth operation requirements

`OAuthOperationScopeProvider` separately declares the OAuth grants for each
registered operation. The registry snapshots these declarations by operation key
and copies both slice levels when returning them. Missing declarations never
grant account-read access. Hosts must independently verify current-user account
ownership, actual grants, the registered operation hash and its read effect;
neither requested scopes nor probe requirements authorize a business operation.

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

## Calendar read contracts

The optional `calendar` package defines a separate `calendar-read-v1` contract
for provider-neutral calendar lists, windowed events, event details and availability.
It includes deterministic operation identities and time validation; it owns no
account, authorization or transport. Date-only all-day bounds remain dates with
exclusive ends. Instants require offsets, and ambiguous local times cannot be
silently selected. Partial availability never implies free time.

## Mail read contracts

The optional `mail` package defines `mail-read-v1` with independent identities
for metadata listing, native-syntax search and bounded plain-text message reads.
Queries declare `gmail` or `graph-kql`; providers reject a mismatched dialect.
Pages default to 10 messages (maximum 25), bodies to 16 KiB (maximum 64 KiB).
IDs are scoped to the authorized account. Missing fields remain unknown;
pagination, provider caps, omitted body content and truncation are explicit.
Attachments and external linked content are outside this read contract.
The package contains no authorization, network, mailbox storage or send API.

The independent `web` package defines `public-web-read-v1` for bounded public-web
search and page reads. Ranked search results never imply an exhaustive match
set; page source completeness remains unknown/partial and local truncation is
explicit. URL normalization checks syntax, common non-public literal addresses,
standard ports and credentials, with no DNS or outbound I/O; hosts and remote
fetch services must enforce network policies separately. Each operation has a
fixed contract hash. The generic Connector contract and calendar/mail identities
remain unchanged.

## Calendar and mail write contracts

`calendarwrite` defines the separate `calendar-write-v1` contract for inspecting
an exact event, creating an event, and applying a version-checked patch. A
single event and a whole recurring series require distinct explicit scopes.
The target attendee list and requested notification policy are part of the
preview; a series can also notify separately changed occurrences. Instants
must have offsets matching their IANA zones, and all-day ends remain exclusive.
Omitted patch fields stay unchanged; explicit empty description/location or
attendee arrays clear those fields. Providers retain their native concurrency
checks and never treat a wildcard as an expected version.

`mailwrite` defines `mail-write-v1` for new mail and replies with explicit To,
CC, BCC, subject and plain text. Replies validate current original metadata and
Reply-To recipients before provider-specific threading. Sender, headers and
correlation identifiers come from the trusted invocation envelope. Receipts
record provider acceptance and explicitly unknown delivery; a Graph 202 needs
no invented message ID.

Both packages reject undeclared request fields and bound JSON requests to
1 MiB, allowing worst-case escaping of bounded 64 KiB text. They do not grant
account access, confirm operations, persist receipts or retry external writes.
Integration and the consuming execution host own those concerns. The calendar
and mail read contracts, existing Provider registrations and generic SDK
contract remain unchanged. Provider and product integration of these optional
contracts is a separate delivery step.
