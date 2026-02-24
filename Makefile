PLUGIN_PREFIX := openbao-plugin
PLUGIN_TYPES := auth secrets database kms
PLUGINS := $(subst /,-,$(wildcard $(foreach t,$(PLUGIN_TYPES),$(t)/*)))
PLUGIN := $(firstword $(PLUGINS))
REGISTRY := ghcr.io/openbao
VERSION := $(shell (git describe --tags --match "$(PLUGIN)-*" 2>/dev/null || echo "v0.0.0") | cut -d- -f3-)

TARGETS := \
    linux_amd64_v1 \
    linux_arm_6 \
    linux_arm64_v8 \
    linux_ppc64le \
    linux_riscv64_rva20u64 \
    linux_s390x \
    darwin_amd64_v1 \
    darwin_arm64_v8 \
    dragonfly_amd64_v1 \
    freebsd_amd64_v1 \
    freebsd_arm_6 \
    freebsd_arm64_v8 \
    illumos_amd64_v1 \
    netbsd_amd64_v1 \
    netbsd_arm_6 \
    netbsd_arm64_v8 \
    openbsd_amd64_v1 \
    openbsd_arm_6 \
    openbsd_arm64_v8 \
    windows_amd64_v1 \
    windows_arm_6 \
    windows_arm64_v8

CGO_PLUGINS := \
	kms-pkcs11

ifneq ($(filter $(PLUGIN),$(CGO_PLUGINS)),)
CGO_ENABLED := 1
# We don't want to modify TARGETS itself such that arbitrary targets can still
# be used with CGO locally, but below are the ones vetted for releases.
CI_TARGETS := \
	linux_amd64_v1 \
	linux_arm64_v8
else
CGO_ENABLED := 0
CI_TARGETS := $(TARGETS)
endif

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
TARGET := $(filter $(GOOS)_$(GOARCH)%,$(TARGETS))

# Define variables based on the target
define set_vars
  ifeq ($1, linux_amd64_v1)
    GOOS := linux
    GOARCH := amd64
    CC := x86_64-linux-gnu-gcc
    CGO_PACKAGES := gcc-x86_64-linux-gnu
  else ifeq ($1, linux_arm_6)
    GOOS := linux
    GOARCH := arm
    GOARM := 6
  else ifeq ($1, linux_arm64_v8)
    GOOS := linux
    GOARCH := arm64
    CC := aarch64-linux-gnu-gcc
    CGO_PACKAGES := gcc-aarch64-linux-gnu
  else ifeq ($1, linux_ppc64le)
    GOOS := linux
    GOARCH := ppc64le
  else ifeq ($1, linux_riscv64_rva20u64)
    GOOS := linux
    GOARCH := riscv64
  else ifeq ($1, linux_s390x)
    GOOS := linux
    GOARCH := s390x
  else ifeq ($1, darwin_amd64_v1)
    GOOS := darwin
    GOARCH := amd64
  else ifeq ($1, darwin_arm64_v8)
    GOOS := darwin
    GOARCH := arm64
  else ifeq ($1, dragonfly_amd64_v1)
    GOOS := dragonfly
    GOARCH := amd64
  else ifeq ($1, freebsd_amd64_v1)
    GOOS := freebsd
    GOARCH := amd64
  else ifeq ($1, freebsd_arm_6)
    GOOS := freebsd
    GOARCH := arm
    GOARM := 6
  else ifeq ($1, freebsd_arm64_v8)
    GOOS := freebsd
    GOARCH := arm64
  else ifeq ($1, illumos_amd64_v1)
    GOOS := illumos
    GOARCH := amd64
  else ifeq ($1, netbsd_amd64_v1)
    GOOS := netbsd
    GOARCH := amd64
  else ifeq ($1, netbsd_arm_6)
    GOOS := netbsd
    GOARCH := arm
    GOARM := 6
  else ifeq ($1, netbsd_arm64_v8)
    GOOS := netbsd
    GOARCH := arm64
  else ifeq ($1, openbsd_amd64_v1)
    GOOS := openbsd
    GOARCH := amd64
  else ifeq ($1, openbsd_arm_6)
    GOOS := openbsd
    GOARCH := arm
    GOARM := 6
  else ifeq ($1, openbsd_arm64_v8)
    GOOS := openbsd
    GOARCH := arm64
  else ifeq ($1, windows_amd64_v1)
    GOOS := windows
    GOARCH := amd64
  else ifeq ($1, windows_arm_6)
    GOOS := windows
    GOARCH := arm
    GOARM := 6
  else ifeq ($1, windows_arm64_v8)
    GOOS := windows
    GOARCH := arm64
  endif
endef

BINARIES := $(addprefix bin/$(PLUGIN_PREFIX)-$(PLUGIN)_,$(TARGETS))
ARCHIVES := $(subst bin/,dist/,$(addsuffix .tar.gz,$(filter-out bin/$(PLUGIN_PREFIX)-$(PLUGIN)_windows%,$(BINARIES))) $(addsuffix .zip,$(filter bin/$(PLUGIN_PREFIX)-$(PLUGIN)_windows%,$(BINARIES))))
SIGNATURES := $(ARCHIVES:=.sig)
SBOMS := $(ARCHIVES:=.spdx.sbom.json)

.PHONY: ci-build-matrix ci-test-matrix ci-targets cgo-packages image push

ci-build-matrix:
	@printf "$(PLUGINS)" | jq -Rscr 'split(" ") | "plugins=\(.)"'

# Save jobs by excluding KMS plugins as their tests are not in this repository.
ci-test-matrix:
	@$(MAKE) --no-print-directory ci-build-matrix PLUGIN_TYPES='auth secrets database'

ci-targets:
	@printf "$(CI_TARGETS)" | jq -Rscr 'split(" ") | "targets=\(.)"'

ifeq ($(CGO_ENABLED),1)
cgo-packages:
	$(eval $(call set_vars,$(TARGET)))
	sudo apt-get install -y $(CGO_PACKAGES)
else
cgo-packages:
endif

bin dist:
	@mkdir -p $@

image: Containerfile $(BINARIES)
	@buildah manifest rm $(REGISTRY)/$(PLUGIN_PREFIX)-$(PLUGIN):$(VERSION) || true
	@buildah manifest create $(REGISTRY)/$(PLUGIN_PREFIX)-$(PLUGIN):$(VERSION)
	@$(foreach target,$(TARGETS),cat $< | PLUGIN=$(PLUGIN) envsubst '$$PLUGIN' | buildah build -f - --platform $(subst _,/,$(target)) --build-arg PLUGIN=$(PLUGIN) -t $(REGISTRY)/$(PLUGIN_PREFIX)-$(PLUGIN):$(VERSION)_$(target);)
	@$(foreach target,$(TARGETS),buildah manifest add $(REGISTRY)/$(PLUGIN_PREFIX)-$(PLUGIN):$(VERSION) $(REGISTRY)/$(PLUGIN_PREFIX)-$(PLUGIN):$(VERSION)_$(target);)

push: image
	@buildah manifest push --all $(REGISTRY)/$(PLUGIN_PREFIX)-$(PLUGIN):$(VERSION)

dist/%.tar.gz: bin/% | dist
	@echo "archiving $@"
	@tar cfz $@ LICENSE -C bin $*

dist/%.zip: bin/% | dist
	@echo "archiving $@"
	@zip -qj $@ LICENSE $^

dist/checksums-$(PLUGIN).txt: $(ARCHIVES) | dist
	@echo "creating checksums $@"
	@cd dist && sha256sum $(subst dist/,,$^) > checksums-$(PLUGIN).txt

dist/%.sig: dist/% | dist
	@echo "signing $@"
	@gpg --detach-sign --pinentry-mode loopback --passphrase $(GPG_PASSWORD) --batch --output $@ $<

dist/%.spdx.sbom.json: dist/% | dist
	@echo "creating SBOM $@"
	@syft scan $< -q -o cyclonedx-json=$@

$(BINARIES): bin/$(PLUGIN_PREFIX)-$(PLUGIN)_%: | bin
	$(eval $(call set_vars,$*))
	@echo "building $@"
	@GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) CGO_ENABLED=$(CGO_ENABLED) CC=$(CC) go build  -o $@ -ldflags '-s -w -X github.com/openbao/openbao-plugins/$(subst -,/,$(PLUGIN)).pluginVersion=$(VERSION)' ./$(subst -,/,$(PLUGIN))/cmd

$(PLUGINS): %:
	@$(MAKE) --no-print-directory build PLUGIN=$*

$(PLUGINS:=-test): %-test:
	@$(MAKE) --no-print-directory test PLUGIN=$*

bin/$(PLUGIN_PREFIX)-$(PLUGIN).test: $(subst -,/,$(PLUGIN))/*.go $(subst -,/,$(PLUGIN))/**/*.go | bin
	@go test -c ./$(subst -,/,$(PLUGIN)) -o $@
	./$@ -test.v -test.short

build: bin/$(PLUGIN_PREFIX)-$(PLUGIN)_$(TARGET)

build-all: $(BINARIES)

release: dist/checksums-$(PLUGIN).txt $(SIGNATURES) $(SBOMS)

test: bin/$(PLUGIN_PREFIX)-$(PLUGIN).test

clean:
	@rm -rf $(TESTS) bin dist
