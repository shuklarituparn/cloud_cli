APP_NAME := cloud_cli
MODULE := github.com/shuklarituparn/cloud_cli
BUILDINFO_PACKAGE := $(MODULE)/internal/buildinfo

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := \
	-X '$(BUILDINFO_PACKAGE).Version=$(VERSION)' \
	-X '$(BUILDINFO_PACKAGE).Commit=$(COMMIT)' \
	-X '$(BUILDINFO_PACKAGE).Date=$(BUILD_DATE)'

.PHONY: build
build:
	mkdir -p bin
	go build \
		-trimpath \
		-ldflags="$(LDFLAGS)" \
		-o bin/$(APP_NAME) \
		./cmd/cloud_cli

.PHONY: release-build
release-build:
	mkdir -p bin
	go build \
		-trimpath \
		-ldflags="-s -w $(LDFLAGS)" \
		-o bin/$(APP_NAME) \
		./cmd/cloud_cli

.PHONY: version-info
version-info:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build date: $(BUILD_DATE)"

.PHONY: docker-build
docker-build:
	docker build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--tag "$(APP_NAME):$(VERSION)" \
		--tag "$(APP_NAME):latest" \
		.

.PHONY: docker-version
docker-version:
	docker run --rm "$(APP_NAME):$(VERSION)" version

.PHONY: test
test:
	go test -race ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofmt -l .)" || \
		(echo "The following files require gofmt:"; gofmt -l .; exit 1)

.PHONY: ci
ci: fmt-check vet test build