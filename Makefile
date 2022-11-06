clean:
	go clean


test:
	go test ./...


test_coverage:
	go test ./... -coverprofile=coverage.out


dep:
	go mod download


vet:
	go vet


lint:
	golangci-lint run --enable-all
