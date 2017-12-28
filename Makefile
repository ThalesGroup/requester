PACKAGES = $$(go list ./... | grep -v /examples)

all: tools ensure fmt vet lint test

test:
	go test $(PACKAGES) -cover

fmt:
	go fmt $(PACKAGES)

lint:
	golint -set_exit_status $(PACKAGES)

vet:
	go vet $(PACKAGES)

tools:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint

ensure:
	dep ensure