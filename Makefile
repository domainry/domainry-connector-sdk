.PHONY: test fmt-check vet boundary

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
