default: build

BINARY=terraform-provider-ractermx
HOSTNAME=registry.terraform.io
NAMESPACE=ractermx
NAME=ractermx
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)
VERSION=0.1.0

build:
	go build -o $(BINARY)

install: build
	mkdir -p ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)
	mv $(BINARY) ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)/

test:
	go test ./... -v $(TESTARGS) -timeout 120s

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

generate:
	go generate ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

.PHONY: build install test testacc generate lint fmt
