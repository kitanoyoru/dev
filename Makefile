.PHONY: gomod lint test

gomod:
	@echo "==> Tidying and vendoring Go modules..."
	go mod tidy
	go mod vendor

lint:
	@echo "==> Running linter..."
	golangci-lint run -c .golangci.yml

test:
	@echo "==> Running tests..."
	gotestsum --format pkgname -- --covermode=atomic --coverpkg=./... --count=1 --race ./...
