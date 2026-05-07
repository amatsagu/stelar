.PHONY: test

STELAR_FILES := $(wildcard test/*.stelar) $(wildcard test/*.stelar.valid)

test:
	@echo "Running Stelar Test Bench..."
	@for file in $(STELAR_FILES); do \
		echo "Testing $$file..."; \
		go run . $$file || exit 1; \
	done
	@echo "All tests passed successfully!"
