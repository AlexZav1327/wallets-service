package models

import (
	"time"

	"github.com/google/uuid"
)

type RequestWalletInstance struct {
	TransactionKey uuid.UUID `json:"transactionKey"`
	WalletID       uuid.UUID `json:"walletId"`
	Email          string    `json:"email"`
	Owner          string    `json:"owner"`
	Currency       string    `json:"currency"`
	Balance        float32   `json:"balance"`
}

type ResponseWalletInstance struct {
	WalletID uuid.UUID `json:"walletId"`
	Email    string    `json:"email"`
	Owner    string    `json:"owner"`
	Currency string    `json:"currency"`
	Balance  float32   `json:"balance"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type FundsOperations struct {
	TransactionKey uuid.UUID `json:"transactionKey"`
	Currency       string    `json:"currency"`
	Amount         float32   `json:"amount"`
}

type RequestWalletHistory struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
	ListingQueryParams
}

type ListingQueryParams struct {
	TextFilter   string
	ItemsPerPage int
	Offset       int
	Sorting      string
	Descending   bool
}

type ResponseWalletHistory struct {
	WalletID  uuid.UUID `json:"walletId"`
	Email     string    `json:"email"`
	Owner     string    `json:"owner"`
	Currency  string    `json:"currency"`
	Balance   float32   `json:"balance"`
	Created   time.Time `json:"created"`
	Operation string    `json:"operation"`
}

type SessionInfo struct {
	UUID  string
	Email string
}
