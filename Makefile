build:
	go build -o ./bin/wallets-service ./cmd/wallets-service/main.go

fmt:
	gofumpt -w .

tidy:	
	go mod tidy

lint: build fmt tidy
	golangci-lint run ./...

run:
	go run ./cmd/wallets-service/main.go

up:
	docker compose up -d

down:
	docker compose down

test: up
	go test -coverpkg=./... -v ./...