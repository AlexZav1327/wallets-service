# Wallets Service
The service is a Go-based application that leverages various technologies to provide wallet management functionalities. 
Using a makefile for streamlined builds, a linter for code quality, and a docker for containerization, the service ensures an effective development environment.
With a focus on transactions, users can create, retrieve, update, and delete wallets, as well as perform deposit, withdrawal, and fund transfer operations. 
The service incorporates secure authentication through JSON Web Tokens (JWT) with Bearer tokens. 
Extensive logging, support for idempotency, integration testing, and detailed metrics contribute to its reliability and maintainability. 
The service uses a PostgreSQL database for efficient and secure data storage.
Additionally, the app is documented with OpenAPI specifications.

## Features
- makefile
- linter
- dockerfile
- transactions
- swagger
- authorization
- logging
- idempotency
- tests
- metrics
- kafka (upcoming change)

## Quick start
Currently, wallets-service requires [Go][go] version 1.21 or greater.

[go]: https://go.dev/doc/install

#### Installation guidelines:
```shell
# Clone a repo
$ git clone https://github.com/AlexZav1327/service.git
# Add missing dependencies
$ go mod tidy
# Start docker containers to launch a PostgreSQL database and an additional exchange rate service
$ make up
# Run server
$ make run
```
#### Integration tests:
```shell
# App, database and migration
$ make test
```
#### Linters:
```shell
# Run linters
$ make lint
```

## API methods description
### Create wallet
```shell
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <user token>" \
  -d '{"email": "sweet-pie@mail.com", "owner": "Liza", "currency": "EUR", "transactionKey": "3a0b5c25-82cc-40ba-a439-81151f7ca695"
  }' \
  'http://localhost:8080/api/v1/wallet/create'
```
#### Response
```json
{"walletId":"c18d130c-245d-44a5-9d1e-9363cc0304d1","email":"sweet-pie@mail.com","owner":"Liza","currency":"EUR","balance":0,"created":"2023-11-27T15:24:22+03:00","updated":"2023-11-27T15:24:22+03:00"}
```
### Get wallet
```shell
curl -X GET \
  -H "Authorization: Bearer <user token>" \
  'http://localhost:8080/api/v1/wallet/c18d130c-245d-44a5-9d1e-9363cc0304d1'
```
#### Response
```json
{"walletId":"c18d130c-245d-44a5-9d1e-9363cc0304d1","email":"sweet-pie@mail.com","owner":"Liza","currency":"EUR","balance":0,"created":"2023-11-27T15:24:22+03:00","updated":"2023-11-27T15:24:22+03:00"}
```
### Get list of wallets
```shell
curl -X GET \
  -H "Authorization: Bearer <user token>" \
  'http://localhost:8080/api/v1/wallets?textFilter=USD&itemsPerPage=2&offset=1&sorting=balance&descending=true'
```
#### Response
```json
[
  {"walletId":"3ced2bb5-a519-44a8-85a2-0c61e17f77d0","email":"hottie@mail.com","owner":"Catherine","currency":"USD","balance":1000,"created":"2023-11-27T16:44:39+03:00","updated":"2023-11-27T16:44:39+03:00"},
  {"walletId":"12bd7b22-a144-4262-9f5c-a07e2ab12fb0","email":"daisy@mail.com","owner":"Catherine","currency":"USD","balance":0,"created":"2023-11-27T16:45:46+03:00","updated":"2023-11-27T16:45:46+03:00"}
]
```
### Get wallet history
```shell
curl -X GET \
  -H "Authorization: Bearer <user token with custom claims (UUID and email)>" \
  'http://localhost:8080/api/v1/wallets??periodStart=2023-11-27T06:59:46&periodEnd=2023-11-27T18:59:46&itemsPerPage=5'
```
#### Response
```json
[
  {"walletId":"4e8db7bd-6d69-4e85-aa4b-888223092969","email":"sweet-pie@mail.com","owner":"Liza","currency":"USD","balance":0,"created":"2023-11-27 19:05:54 +0300 MSK","operation":"CREATE"},
  {"walletId":"4e8db7bd-6d69-4e85-aa4b-888223092969","email":"sweet-pie@mail.com","owner":"Liza","currency":"USD","balance":1000,"created":"2023-11-27 19:06:38 +0300 MSK","operation":"UPDATE"},
  {"walletId":"4e8db7bd-6d69-4e85-aa4b-888223092969","email":"sweet-pie@mail.com","owner":"Liza","currency":"USD","balance":850,"created":"2023-11-27 19:11:51 +0300 MSK","operation":"UPDATE"},
  {"walletId":"4e8db7bd-6d69-4e85-aa4b-888223092969","email":"sweet-pie@mail.com","owner":"Liza","currency":"EUR","balance":782,"created":"2023-11-27 19:16:04 +0300 MSK","operation":"UPDATE"},
  {"walletId":"4e8db7bd-6d69-4e85-aa4b-888223092969","email":"sweet-pie@mail.com","owner":"Liza","currency":"EUR","balance":782,"created":"2023-11-27 19:55:12 +0300 MSK","operation":"DELETE"}
]
```
### Update wallet
```shell
curl -X PATCH \
  -H "Authorization: Bearer <user token>" \
  -H "Content-Type: application/json" \
  -d '{
    "email":"duchess@mail.com", 
    "owner": "Liza Zav", 
    "currency": "USD"
  }' \
  'http://localhost:8080/api/v1/wallet/update/3ced2bb5-a519-44a8-85a2-0c61e17f77d0'
```
#### Response
```json
{"walletId":"3ced2bb5-a519-44a8-85a2-0c61e17f77d0","email":"duchess@mail.com","owner":"Liza Zav","currency":"USD","balance":0,"created":"2023-11-27T16:44:39+03:00","updated":"2023-11-27T17:38:19.781504+03:00"}
```
### Delete wallet
```shell
curl -X DELETE \
-H "Authorization: Bearer <user token>" \
'http://localhost:8080/api/v1/wallet/delete/12bd7b22-a144-4262-9f5c-a07e2ab12fb0'
```
### Deposit funds
```shell
curl -X PUT \
  -H "Authorization: Bearer <user token>" \
  -H "Content-Type: application/json" \
  -d '{
    "currency": "EUR", 
    "amount": 1000, 
    "transactionKey": "ee8c30b6-0fb9-423b-8f17-03602a29307a"
  }' \
  'http://localhost:8080/api/v1/wallet/3ced2bb5-a519-44a8-85a2-0c61e17f77d0/deposit'
```
#### Response
```json
{"walletId":"3ced2bb5-a519-44a8-85a2-0c61e17f77d0","email":"duchess@mail.com","owner":"Liza Zav","currency":"USD","balance":1070,"created":"2023-11-27T16:44:39+03:00","updated":"2023-11-27T18:10:28.580576+03:00"}
```
### Withdraw funds
```shell
curl -X PUT \
  -H "Authorization: Bearer <user token>" \
  -H "Content-Type: application/json" \
  -d '{
    "currency": "RUB", 
    "amount": 5000, 
    "transactionKey": "169c20f5-d5b3-4520-a99c-172da0e1d5b5"
  }' \
  'http://localhost:8080/api/v1/wallet/3ced2bb5-a519-44a8-85a2-0c61e17f77d0/withdraw'
```
#### Response
```json
{"walletId":"3ced2bb5-a519-44a8-85a2-0c61e17f77d0","email":"duchess@mail.com","owner":"Liza Zav","currency":"USD","balance":1019,"created":"2023-11-27T16:44:39+03:00","updated":"2023-11-27T18:14:47.36418+03:00"}

```
### Transfer funds
```shell
curl -X PUT \
  -H "Authorization: Bearer <user token>" \
  -H "Content-Type: application/json" \
  -d '{
    "currency": "USD", 
    "amount": 600, 
    "transactionKey": "d2a08294-a0af-478e-b4b2-a77f24e57c55"
  }' \
  'http://localhost:8080/api/v1/wallet/3ced2bb5-a519-44a8-85a2-0c61e17f77d0/transfer/7bad323e-f0fd-4eeb-80ff-5dfd95bd66c5'
```
#### Response
```json
{"walletId":"7bad323e-f0fd-4eeb-80ff-5dfd95bd66c5","email":"hottie@mail.com","owner":"Catherine","currency":"EUR","balance":552,"created":"2023-11-27T18:18:52+03:00","updated":"2023-11-27T18:28:51.658259+03:00"}
```
