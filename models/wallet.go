package models

import (
	"time"

	"github.com/google/uuid"
)

type WalletInstance struct {
	WalletID uuid.UUID `json:"walletId"`
	Owner    string    `json:"owner"`
	Balance  float32   `json:"balance"`
	Currency string    `json:"currency"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type ChangeWalletData struct {
	WalletID uuid.UUID `json:"walletId"`
	Owner    string    `json:"owner"`
	Balance  float32   `json:"balance"`
}

type WrongWalletData struct {
	WalletID uuid.UUID `json:"walletId"`
	Balance  string    `json:"balance"`
}

type Overdraft struct {
	WalletID uuid.UUID `json:"walletId"`
	Balance  float32   `json:"balance"`
}
