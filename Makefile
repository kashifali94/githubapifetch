APP_NAME=github-fetch
TEST_BINARY=$(APP_NAME).test

.PHONY: build test test-binary clean

build:
	@echo "🔧 Building main binary..."
	GOOS=linux GOARCH=amd64 go build -o $(APP_NAME) main.go

# Build test binaries for each package
test-binaries:
	@echo "🧪 Building test binaries for each package..."
	@mkdir -p test-binaries
	@for pkg in $$(go list ./...); do \
		name=$$(echo $$pkg | sed 's|/|_|g'); \
		echo "🔨 Building test binary for $$pkg -> test-binaries/$$name.test"; \
		GOOS=linux GOARCH=amd64 go test -coverpkg=./... -c -o test-binaries/$$name.test $$pkg || echo "⚠️  Skipped $$pkg (no tests)"; \
	done

test-binary:
	@echo "🧪 Building test binary for the entire project..."
	GOOS=linux GOARCH=amd64 go test -coverpkg=./... -c -o $(TEST_BINARY)



clean:
	@echo "🧹 Cleaning up..."
	rm -f $(APP_NAME) $(TEST_BINARY)
