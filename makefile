BUILD_OUTPUT=build
CUSTOM=-X 'main.buildDate=$(shell date)' -X 'main.gitHash=$(shell git rev-parse --short HEAD)' -X 'main.buildOn=$(shell go version)'
LDFLAGS=$(CUSTOM) -w -s -extldflags=-static
GO_BUILD=go build -trimpath -ldflags "$(LDFLAGS)"

APP_PATH=./src
APP_NAME=archivemapper

define GO_BUILD_CMD
	CGO_ENABLED=1 GOOS=$(1) GOARCH=$(2) $(GO_BUILD) -o $(BUILD_OUTPUT)/$(3) $(4)
endef

.PHONY: fmt
fmt:
	gofumpt -l -w -extra .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint: fmt
	golangci-lint run ./... --fix

.PHONY: build-all
build-all: build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows

.PHONY: build-linux
build-linux: fmt
	$(call GO_BUILD_CMD,linux,amd64,$(APP_NAME)-linux,$(APP_PATH))

.PHONY: build-linux-arm64
build-linux-arm64: fmt
	$(call GO_BUILD_CMD,linux,arm64,$(APP_NAME)-linux-arm64,$(APP_PATH))

.PHONY: build-darwin
build-darwin: fmt
	$(call GO_BUILD_CMD,darwin,amd64,$(APP_NAME)-darwin,$(APP_PATH))

.PHONY: build-darwin-arm64
build-darwin-arm64: fmt
	$(call GO_BUILD_CMD,darwin,arm64,$(APP_NAME)-darwin-arm64,$(APP_PATH))

.PHONY: build-windows
build-windows: fmt
	$(call GO_BUILD_CMD,windows,amd64,$(APP_NAME).exe,$(APP_PATH))
