package models

import (
	"time"

	"github.com/google/uuid"
)

type WalletInstance struct {
	WalletID uuid.UUID `json:"walletId"`
	Owner    string    `json:"owner"`
	Currency string    `json:"currency"`
	Balance  float32   `json:"balance"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type ChangeCurrency struct {
	Currency string `json:"currency"`
}

type ManageFunds struct {
	Currency string  `json:"currency"`
	Amount   float32 `json:"amount"`
}
