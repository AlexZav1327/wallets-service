package models

import (
	"time"

	"github.com/google/uuid"
)

type WalletData struct {
	WalletID uuid.UUID `json:"walletId"`
	Owner    *string   `json:"owner"`
	Balance  *float32  `json:"balance"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}
