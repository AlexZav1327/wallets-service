package xrservice

import (
	"errors"
	"time"

	"github.com/AlexZav1327/service/models"
	"github.com/sirupsen/logrus"
)

const (
	eur                 = "EUR"
	rub                 = "RUB"
	usd                 = "USD"
	ask                 = "ask"
	bid                 = "bid"
	eurToRubBid float32 = 99.35
	eurToRubAsk float32 = 105.50
	eurToUsdBid float32 = 1.07
	eurToUsdAsk float32 = 1.13
	rubToEurBid float32 = 0.0095
	rubToEurAsk float32 = 0.0101
	rubToUsdBid float32 = 0.0102
	rubToUsdAsk float32 = 0.0108
	usdToEurBid float32 = 0.92
	usdToEurAsk float32 = 0.97
	usdToRubBid float32 = 92.22
	usdToRubAsk float32 = 98.19
)

var ErrWrongCurrency = errors.New("currency is not valid")

type Rate struct {
	log *logrus.Entry
}

func New(log *logrus.Logger) *Rate {
	return &Rate{
		log: log.WithField("module", "xr_service"),
	}
}

func (r *Rate) GetCurrentRate(from, to string) (models.ExchangeRate, error) {
	var currentRate models.ExchangeRate

	var currencyRate map[string]float32

	var ok bool

	currentRate.Timestamp = time.Now()

	rates := map[string]map[string]map[string]float32{
		eur: {rub: {bid: eurToRubBid, ask: eurToRubAsk}, usd: {bid: eurToUsdBid, ask: eurToUsdAsk}},
		rub: {eur: {bid: rubToEurBid, ask: rubToEurAsk}, usd: {bid: rubToUsdBid, ask: rubToUsdAsk}},
		usd: {eur: {bid: usdToEurBid, ask: usdToEurAsk}, rub: {bid: usdToRubBid, ask: usdToRubAsk}},
	}

	if rates[from] != nil {
		currencyRate, ok = rates[from][to]
		if !ok {
			return models.ExchangeRate{}, ErrWrongCurrency
		}
	} else {
		return models.ExchangeRate{}, ErrWrongCurrency
	}

	currentRate.Bid = currencyRate[bid]
	currentRate.Ask = currencyRate[ask]
	currentRate.Currencies = from + to

	return currentRate, nil
}
