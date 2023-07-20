build:
	go build -o bin/service cmd/server/main.go

fmt:
	gofumpt -w .

tidy:	
	go mod tidy

lint: build fmt tidy
	golangci-lint run ./...

run:
	go run cmd/server/main.go