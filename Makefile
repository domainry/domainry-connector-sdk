.PHONY: test fmt-check vet boundary license-check vulnerability-check release-check

test:
	go test ./...

fmt-check:
	@files="$$(find . -name '*.go' -not -path './.git/*' -print | xargs gofmt -l)"; \
	if [ -n "$$files" ]; then \
		printf 'gofmt needed:\n%s\n' "$$files" >&2; \
		exit 1; \
	fi

vet:
	go vet ./...

boundary:
	@deps="$$(go list -deps ./...)"; \
	if printf '%s\n' "$$deps" | grep -Eq '^github\.com/domainry/(domainry-plane|domainry-connectors)(/|$$)'; then \
		printf 'forbidden module dependency detected\n' >&2; \
		exit 1; \
	fi

license-check:
	@grep -qx 'MIT License' LICENSE
	@grep -qx 'Copyright (c) 2026 Domainry' LICENSE

vulnerability-check:
	GOTOOLCHAIN=go1.26.6 go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

release-check: fmt-check test vet boundary license-check vulnerability-check
