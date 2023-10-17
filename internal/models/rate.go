package models

import "time"

type ExchangeRate struct {
	Timestamp  time.Time `json:"timestamp"`
	Currencies string    `json:"currencies"`
	Bid        float32   `json:"bid"`
	Ask        float32   `json:"ask"`
}
