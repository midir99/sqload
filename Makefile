.PHONY: help clean fmt test coverage coverage-html dep vet lint

.DEFAULT_GOAL := help

## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' |  sed -e 's/^/ /'


## clean: removes object files from source directories
clean:
	go clean


## fmt: formats the code using golangci-lint
fmt:
	golangci-lint fmt


## test: runs the tests
test:
	go test ./...


## coverage: runs the tests and reports the test coverage
coverage:
	go test ./... -coverprofile=coverage.out


## coverage-html: runs the tests and reports the test coverage in html format
coverage-html: coverage
	go tool cover -html=coverage.out


## dep: downloads the dependencies
dep:
	go mod download


## vet: runs the command go vet
vet:
	go vet ./...


## lint: lints the code using golangci-lint
lint:
	golangci-lint run
