PACKAGES = $$(go list ./...)

all: tools ensure fmt lint test

test:
	go test $(PACKAGES)

build:
	mkdir build

coverage: build
	go test $(PACKAGES) -covermode=count -coverprofile=build/coverage.out
	go tool cover -html=build/coverage.out -o build/coverage.html

fmt:
	go fmt $(PACKAGES)


lint:
	# running golangci-lint run twice, ignoring the first run
	# this is to workaround a bug in golangci-lint's type cache which can result
	# in false type errors on the first run
	# https://github.com/golangci/golangci-lint/issues/885
	-golangci-lint run > /dev/null 2>&1
	golangci-lint run

vet:
	go vet $(PACKAGES)

tools:
#	go get -u github.com/golang/lint/golint
	sh -c "$$(wget -O - -q https://install.goreleaser.com/github.com/golangci/golangci-lint.sh || echo exit 2)" -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

ensure:
	dep ensure