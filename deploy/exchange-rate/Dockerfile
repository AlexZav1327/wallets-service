FROM golang:latest as builder
ADD . /src/xr
WORKDIR /src/xr
RUN CGO_ENABLED=0 GOOS=linux go build -o exchange-rate ./cmd/xr-stub/main.go
EXPOSE 8091

FROM alpine:edge
COPY --from=builder /src/xr/exchange-rate /exchange-rate
RUN chmod +x ./exchange-rate
ENTRYPOINT ["/exchange-rate"]
