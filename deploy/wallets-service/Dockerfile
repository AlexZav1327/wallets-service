FROM golang:latest as builder
ADD . /src/app
WORKDIR /src/app
RUN CGO_ENABLED=0 GOOS=linux go build -o wallets-service ./cmd/wallets-service/main.go
EXPOSE 8080

FROM alpine:edge
COPY --from=builder /src/app/wallets-service /wallets-service
RUN chmod +x ./wallets-service
ENTRYPOINT ["/wallets-service"]