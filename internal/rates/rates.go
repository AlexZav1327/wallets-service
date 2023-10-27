package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/sirupsen/logrus"
)

type Rates struct {
	log     *logrus.Entry
	metrics *metrics
}

func New(log *logrus.Logger) *Rates {
	return &Rates{
		log:     log.WithField("module", "rates"),
		metrics: newMetrics(),
	}
}

func (r *Rates) GetRate(ctx context.Context, currentCurrency, requestedCurrency string) (
	models.ExchangeRate, error,
) {
	started := time.Now()
	defer func() {
		r.metrics.duration.Observe(time.Since(started).Seconds())
	}()

	endpoint := fmt.Sprintf("http://localhost:8091/api/v1/xr?from=%s&to=%s", currentCurrency, requestedCurrency)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return models.ExchangeRate{}, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return models.ExchangeRate{}, fmt.Errorf("http.DefaultClient.Do: %w", err)
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			r.log.Warningf("resp.Body.Close: %s", err)
		}
	}()

	var rates models.ExchangeRate

	err = json.NewDecoder(response.Body).Decode(&rates)
	if err != nil {
		return models.ExchangeRate{}, fmt.Errorf("json.NewDecoder.Decode: %w", err)
	}

	return rates, nil
}
