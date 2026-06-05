default: test

.PHONY: build test testacc fmt vet tidy

build:
	go build ./...

test:
	go test ./internal/... ./...

testacc:
	TF_ACC=1 go test ./internal/provider/... -v -timeout 120m

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy
