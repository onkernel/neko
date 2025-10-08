SHELL := /bin/bash
.PHONY: generate-client install-tools clean-client help

BIN_DIR ?= $(CURDIR)/bin
OAPI_CODEGEN ?= $(BIN_DIR)/oapi-codegen

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(OAPI_CODEGEN): | $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

generate-client: $(OAPI_CODEGEN)
	@echo "Generating Go client from OpenAPI spec..."
	mkdir -p lib/oapi
	$(OAPI_CODEGEN) -config oapi-codegen.client.yaml openapi.yaml
	@echo "Formatting generated code..."
	go fmt ./lib/oapi/...
	@echo "Client generated at ./lib/oapi/"

install-tools: $(OAPI_CODEGEN)
	@echo "Tools installed to $(BIN_DIR)"

clean-client:
	rm -rf lib/oapi/client.go

help:
	@echo "Available targets:"
	@echo "  generate-client  - Generate Go client from OpenAPI spec"
	@echo "  install-tools    - Install required tools"
	@echo "  clean-client     - Remove generated client code"

