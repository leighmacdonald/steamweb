all: fmt check
	go build

fmt:
	gci write . --skip-generated -s default
	gofumpt -l -w .

check: lint_golangci static

lint_golangci:
	@golangci-lint run --timeout 3m

static:
	@staticcheck -go 1.20 ./...

check_deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
	go install github.com/daixiang0/gci@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

test:
	go test ./...

update:
	go get -u ./...
