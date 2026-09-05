GOLANGCI_LINT_VERSION ?= v2.13.2
GOBIN ?= $(shell go env GOPATH)/bin
GOLANGCI_LINT := $(GOBIN)/golangci-lint

.PHONY: test view lint lint-fix lint-install

test:
	go test -coverprofile=coverage.out ./...

view:
	go tool cover -html=coverage.out

# Installs golangci-lint only if it is missing or the wrong version.
# It is a tool, not a library dependency, so it never enters go.mod.
lint-install:
	@if ! "$(GOLANGCI_LINT)" --version 2>/dev/null | grep -q "$(GOLANGCI_LINT_VERSION:v%=%)"; then \
		echo "installing golangci-lint $(GOLANGCI_LINT_VERSION)"; \
		GOBIN="$(GOBIN)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

lint: lint-install
	"$(GOLANGCI_LINT)" run ./...

lint-fix: lint-install
	"$(GOLANGCI_LINT)" run --fix ./...
