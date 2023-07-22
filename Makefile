build:
	go build -o bin/service cmd/base-point/main.go

fmt:
	gofumpt -w .

tidy:	
	go mod tidy

lint: build fmt tidy
	golangci-lint run ./...

run:
	go run cmd/base-point/main.go