build:
	go build -o bin/service cmd/app/main.go

fmt:
	gofumpt -w .

tidy:	
	go mod tidy

lint: build fmt tidy
	golangci-lint run ./...

run:
	go run cmd/app/main.go

up:
	docker compose up -d

down:
	docker compose down

test:
	go test -v ./...