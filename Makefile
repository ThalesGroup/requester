PACKAGES = $$(go list ./...)

all: tools ensure fmt vet lint test

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
	golint -set_exit_status $(PACKAGES)

vet:
	go vet $(PACKAGES)

tools:
	go get -u github.com/golang/lint/golint

ensure:
	dep ensure