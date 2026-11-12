##@ Security

.PHONY: security

## Run security analysis with gosec
security: $(GOSEC)
	@$(GOSEC) -stdout -severity medium ./...
