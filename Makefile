.PHONY: test test-race coverage benchmark clean coverage-clean go-clean

COVERAGE_DIR := coverage

test:
	go test ./...

test-race:
	go test -race -v -run=^TestRace.*$$ ./...

coverage:
	mkdir -p $(COVERAGE_DIR)
	go test -covermode=count -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

benchmark:
	go test -benchmem -run=^$$ -bench=^Benchmark.*$

clean: coverage-clean go-clean
	rm -rf bin

coverage-clean:
	rm -rf $(COVERAGE_DIR)

go-clean: 
	go clean -testcache
