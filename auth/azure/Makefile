REPO_DIR := $(shell basename $(CURDIR))
PLUGIN_NAME := openbao-plugin-auth-azure
PLUGIN_DIR ?= $$GOPATH/openbao-plugins
PLUGIN_PATH ?= local-auth-azure

# bin generates the releasable binaries for this plugin
.PHONY: bin
bin:
	@CGO_ENABLED=0 BUILD_TAGS='$(BUILD_TAGS)' sh -c "'$(CURDIR)/scripts/build.sh'"

.PHONY: default
default: dev

.PHONY: dev
dev:
	@CGO_ENABLED=0 BUILD_TAGS='$(BUILD_TAGS)' VAULT_DEV_BUILD=1 sh -c "'$(CURDIR)/scripts/build.sh'"

.PHONY: bootstrap
bootstrap:
	@echo "Downloading tools ..."
	@go generate -tags tools tools/tools.go

.PHONY: test
test: fmtcheck
	CGO_ENABLED=0 go test ./... $(TESTARGS) -timeout=20m

.PHONY: fmtcheck
fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

.PHONY: fmt
fmt:
	gofumpt -l -w .

.PHONY: setup-env
setup-env:
	cd bootstrap/terraform && terraform init && terraform apply -auto-approve

.PHONY: teardown-env
teardown-env:
	cd bootstrap/terraform && terraform init && terraform destroy -auto-approve

.PHONY: configure
configure: dev
	@./bootstrap/configure.sh \
	$(PLUGIN_DIR) \
	$(PLUGIN_NAME) \
	$(PLUGIN_PATH)
