all: tidy fmt lint test

test:
	go test ./...

testall:
	go test -count 1 ./...

build:
	mkdir build

cover: build
	go test ./... -covermode=count -coverprofile=build/coverage.out
	go tool cover -html=build/coverage.out -o build/coverage.html

fmt:
	go fmt ./...

lint:
	golangci-lint run

tools:
	sh -c "$$(wget -O - -q https://install.goreleaser.com/github.com/golangci/golangci-lint.sh || echo exit 2)" -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

tidy:
	go mod tidy

# use go mod to update all dependencies
update:
	go get -u ./...
	go mod tidy