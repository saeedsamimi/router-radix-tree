.PHONY: test coverage benchmark clean coverage-clean go-clean

COVERAGE_DIR := coverage

# OS detection
ifeq ($(OS),Windows_NT)
	RM := del /Q /S
	RMDIR := rmdir /Q /S
	MKDIR := mkdir
	SEP := \\
else
	RM := rm -f
	RMDIR := rm -rf
	MKDIR := mkdir -p
	SEP := /
endif

test:
	go test ./...

coverage:
	$(MKDIR) $(COVERAGE_DIR)
	go test -covermode=count -coverprofile=$(COVERAGE_DIR)$(SEP)coverage.out ./...
	go tool cover -html=$(COVERAGE_DIR)$(SEP)coverage.out -o $(COVERAGE_DIR)$(SEP)coverage.html
	@echo "Coverage report generated at $(COVERAGE_DIR)$(SEP)coverage.html"

benchmark:
	go test -benchmem -run=^$$ -bench=^Benchmark.*$

clean: coverage-clean go-clean
	$(RMDIR) bin

coverage-clean:
	$(RMDIR) $(COVERAGE_DIR)

go-clean: 
	go clean -testcache
