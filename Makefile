PACKAGES = $$(go list ./... | grep -v /examples)

all: fmt vet lint test

test:
	go test $(PACKAGES) -cover

fmt:
	go fmt $(PACKAGES)

lint:
	golint -set_exit_status $(PACKAGES)

vet:
	go vet $(PACKAGES)