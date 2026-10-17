.PHONY: security govulncheck sbom vuln

## Run security analysis with gosec
security: $(GOSEC)
	@$(GOSEC) -stdout -severity medium ./...

# gosec reads this repository's code; govulncheck reads the dependency tree the
# way the compiler does, so it only reports advisories whose vulnerable symbol
# is actually reachable from this module. That is the difference between "a CVE
# exists somewhere in go.sum" and "this binary can execute it".

## Report vulnerabilities reachable from this code
govulncheck: $(GOVULNCHECK)
	@echo "Checking for reachable vulnerabilities..."
	@$(GOVULNCHECK) ./...

# Double quotes: cmd.exe keeps single quotes as part of the argument.
## Generate a CycloneDX SBOM of the dependency tree
sbom: $(SYFT)
	@echo "Generating SBOM..."
	@$(SYFT) . --source-name $(BINARY_NAME) --exclude "./.gobin/**" --exclude "./$(BUILD_DIR)/**" --exclude "./dist/**" -o cyclonedx-json=$(SBOM_FILE)

## Scan the SBOM for known vulnerabilities with grype
vuln: $(GRYPE) sbom
	@echo "Scanning for vulnerabilities..."
	@$(GRYPE) sbom:$(SBOM_FILE) --fail-on medium
