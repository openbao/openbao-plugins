PLUGIN_PREFIX := openbao-plugin
PLUGINS := $(subst /,-,$(wildcard auth/* secrets/* databases/*))
PLUGIN := $(firstword $(PLUGINS))

.PHONY: ci-matrix

ci-matrix:
	@echo -n "$(PLUGINS)"  | jq -Rscr 'split(" ") | "plugins=\(.)"'

bin:
	@mkdir -p bin

$(PLUGINS): %:
	@$(MAKE) --no-print-directory build PLUGIN=$*

$(PLUGINS:=-test): %-test:
	@$(MAKE) --no-print-directory test PLUGIN=$*

bin/$(PLUGIN_PREFIX)-$(PLUGIN): | bin
	@goreleaser build --single-target -o $@ --id $(PLUGIN) --snapshot --clean
	@rm -rf dist

bin/$(PLUGIN_PREFIX)-$(PLUGIN).test: $(subst -,/,$(PLUGIN))/*.go $(subst -,/,$(PLUGIN))/**/*.go | bin
	@go test -c ./$(subst -,/,$(PLUGIN)) -o $@
	./$@ -test.v -test.short

build: bin/$(PLUGIN_PREFIX)-$(PLUGIN)

test: bin/$(PLUGIN_PREFIX)-$(PLUGIN).test

clean:
	@rm -rf $(TESTS) bin dist
