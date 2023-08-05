FROM golang:latest as builder
ADD . /src/app
WORKDIR /src/app
RUN CGO_ENABLED=0 GOOS=linux go build -o web-service cmd/base-point/main.go
EXPOSE 8080

FROM alpine:edge
COPY --from=builder /src/app/web-service /web-service
RUN chmod +x ./web-service
ENTRYPOINT ["/web-service"]
