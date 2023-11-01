package walletservice

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	eur = "EUR"
	rub = "RUB"
	usd = "USD"
)

var (
	ErrOverdraft        = errors.New("overdrafts are not allowed")
	ErrCurrencyNotValid = errors.New("currency is not valid")
)

type Service struct {
	pg      WalletStore
	xr      ExchangeRates
	log     *logrus.Entry
	metrics *metrics
}

type WalletStore interface {
	CreateWallet(ctx context.Context, wallet models.RequestWalletInstance) (models.ResponseWalletInstance, error)
	GetWalletsList(ctx context.Context, params models.ListingQueryParams) ([]models.ResponseWalletInstance, error)
	GetWallet(ctx context.Context, id string) (models.ResponseWalletInstance, error)
	GetWalletHistory(ctx context.Context, id string, params models.RequestWalletHistory) (
		[]models.ResponseWalletHistory, error)
	UpdateWallet(ctx context.Context, wallet models.RequestWalletInstance) (models.ResponseWalletInstance, error)
	DeleteWallet(ctx context.Context, id string) error
	ManageBalance(ctx context.Context, transactionKey uuid.UUID, id string, balance float32) (
		models.ResponseWalletInstance, error)
	TransferFunds(ctx context.Context, transactionKey uuid.UUID, idSrc, idDst string, balanceSrc, balanceDst float32) (
		models.ResponseWalletInstance, error)
}

type ExchangeRates interface {
	GetRate(ctx context.Context, currentCurrency, requestedCurrency string) (models.ExchangeRate, error)
}

func New(pg WalletStore, xr ExchangeRates, log *logrus.Logger) *Service {
	return &Service{
		pg:      pg,
		xr:      xr,
		log:     log.WithField("module", "service"),
		metrics: newMetrics(),
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet models.RequestWalletInstance) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(wallet.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrCurrencyNotValid
	}

	started := time.Now()
	defer func() {
		s.metrics.duration.WithLabelValues("create_wallet").Observe(time.Since(started).Seconds())
	}()

	createdWallet, err := s.pg.CreateWallet(ctx, wallet)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.CreateWallet: %w", err)
	}

	s.metrics.wallets.Inc()

	return createdWallet, nil
}

func (s *Service) GetWalletsList(ctx context.Context, params models.ListingQueryParams) (
	[]models.ResponseWalletInstance, error,
) {
	walletsList, err := s.pg.GetWalletsList(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("pg.GetWalletsList: %w", err)
	}

	return walletsList, nil
}

func (s *Service) GetWallet(ctx context.Context, id string) (models.ResponseWalletInstance, error) {
	started := time.Now()
	defer func() {
		s.metrics.duration.WithLabelValues("get_wallet").Observe(time.Since(started).Seconds())
	}()

	wallet, err := s.pg.GetWallet(ctx, id)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.GetWallet: %w", err)
	}

	return wallet, nil
}

func (s *Service) GetWalletHistory(ctx context.Context, id string, params models.RequestWalletHistory) (
	[]models.ResponseWalletHistory, error,
) {
	walletHistory, err := s.pg.GetWalletHistory(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("pg.GetWalletHistory: %w", err)
	}

	return walletHistory, nil
}

func (s *Service) UpdateWallet(ctx context.Context, wallet models.RequestWalletInstance) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(wallet.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrCurrencyNotValid
	}

	currentWallet, err := s.pg.GetWallet(ctx, wallet.WalletID.String())
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.GetWallet: %w", err)
	}

	if wallet.Owner == "" {
		wallet.Owner = currentWallet.Owner
	}

	if wallet.Currency == currentWallet.Currency || wallet.Currency == "" {
		wallet.Balance = currentWallet.Balance
	} else {
		wallet.Balance, err = s.ConvertCurrency(ctx, currentWallet.Currency, wallet.Currency, currentWallet.Balance)
		if err != nil {
			return models.ResponseWalletInstance{}, fmt.Errorf("ConvertCurrency: %w", err)
		}
	}

	started := time.Now()
	defer func() {
		s.metrics.duration.WithLabelValues("update_wallet").Observe(time.Since(started).Seconds())
	}()

	updatedWallet, err := s.pg.UpdateWallet(ctx, wallet)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.UpdateWallet: %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) DeleteWallet(ctx context.Context, id string) error {
	started := time.Now()
	defer func() {
		s.metrics.duration.WithLabelValues("delete_wallet").Observe(time.Since(started).Seconds())
	}()

	err := s.pg.DeleteWallet(ctx, id)
	if err != nil {
		return fmt.Errorf("pg.DeleteWallet: %w", err)
	}

	s.metrics.deletedWallets.Inc()

	return nil
}

func (s *Service) DepositFunds(ctx context.Context, id string, depositFunds models.FundsOperations) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(depositFunds.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrCurrencyNotValid
	}

	currentWallet, err := s.pg.GetWallet(ctx, id)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.GetWallet: %w", err)
	}

	balance := currentWallet.Balance + depositFunds.Amount

	if depositFunds.Currency != currentWallet.Currency {
		convertedDepositAmount, err := s.ConvertCurrency(
			ctx,
			depositFunds.Currency,
			currentWallet.Currency,
			depositFunds.Amount,
		)
		if err != nil {
			return models.ResponseWalletInstance{}, fmt.Errorf("ConvertCurrency: %w", err)
		}

		balance = currentWallet.Balance + convertedDepositAmount
	}

	started := time.Now()
	defer func() {
		s.metrics.duration.WithLabelValues("deposit").Observe(time.Since(started).Seconds())
	}()

	updatedWallet, err := s.pg.ManageBalance(ctx, depositFunds.TransactionKey, id, balance)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.ManageFunds: %w", err)
	}

	s.metrics.funds.WithLabelValues(depositFunds.Currency).Add(float64(depositFunds.Amount))

	return updatedWallet, nil
}

func (s *Service) WithdrawFunds(ctx context.Context, id string, withdrawFunds models.FundsOperations) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(withdrawFunds.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrCurrencyNotValid
	}

	currentWallet, err := s.pg.GetWallet(ctx, id)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.GetWallet: %w", err)
	}

	balance := currentWallet.Balance - withdrawFunds.Amount

	if withdrawFunds.Currency != currentWallet.Currency {
		convertedWithdrawAmount, err := s.ConvertCurrency(
			ctx,
			withdrawFunds.Currency,
			currentWallet.Currency,
			withdrawFunds.Amount,
		)
		if err != nil {
			return models.ResponseWalletInstance{}, fmt.Errorf("ConvertCurrency: %w", err)
		}

		balance = currentWallet.Balance - convertedWithdrawAmount
	}

	if balance < 0 {
		return models.ResponseWalletInstance{}, ErrOverdraft
	}

	started := time.Now()
	defer func() {
		s.metrics.duration.WithLabelValues("withdraw").Observe(time.Since(started).Seconds())
	}()

	updatedWallet, err := s.pg.ManageBalance(ctx, withdrawFunds.TransactionKey, id, balance)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.ManageFunds: %w", err)
	}

	s.metrics.funds.WithLabelValues(withdrawFunds.Currency).Sub(float64(withdrawFunds.Amount))

	return updatedWallet, nil
}

func (s *Service) TransferFunds(ctx context.Context, idSrc, idDst string, transferFunds models.FundsOperations) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(transferFunds.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrCurrencyNotValid
	}

	currentSrcWallet, err := s.pg.GetWallet(ctx, idSrc)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.GetWallet: %w", err)
	}

	balanceSrc := currentSrcWallet.Balance - transferFunds.Amount

	if transferFunds.Currency != currentSrcWallet.Currency {
		convertedWithdrawAmount, err := s.ConvertCurrency(ctx, transferFunds.Currency, currentSrcWallet.Currency,
			transferFunds.Amount,
		)
		if err != nil {
			return models.ResponseWalletInstance{}, fmt.Errorf("ConvertCurrency: %w", err)
		}

		balanceSrc = currentSrcWallet.Balance - convertedWithdrawAmount
	}

	if balanceSrc < 0 {
		return models.ResponseWalletInstance{}, ErrOverdraft
	}

	currentDstWallet, err := s.pg.GetWallet(ctx, idDst)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.GetWallet: %w", err)
	}

	balanceDst := currentDstWallet.Balance + transferFunds.Amount

	if transferFunds.Currency != currentDstWallet.Currency {
		convertedDepositAmount, err := s.ConvertCurrency(ctx, transferFunds.Currency, currentDstWallet.Currency,
			transferFunds.Amount,
		)
		if err != nil {
			return models.ResponseWalletInstance{}, fmt.Errorf("ConvertCurrency: %w", err)
		}

		balanceDst = currentDstWallet.Balance + convertedDepositAmount
	}

	started := time.Now()
	defer func() {
		s.metrics.duration.WithLabelValues("transfer").Observe(time.Since(started).Seconds())
	}()

	updatedWallet, err := s.pg.TransferFunds(ctx, transferFunds.TransactionKey, idSrc, idDst, balanceSrc, balanceDst)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.TransferFunds: %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) ConvertCurrency(ctx context.Context, currentCurrency, requestedCurrency string,
	currentBalance float32,
) (float32, error) {
	rates, err := s.xr.GetRate(ctx, currentCurrency, requestedCurrency)
	if err != nil {
		return 0, fmt.Errorf("xr.GetRate: %w", err)
	}

	convertedBalance := float32(math.Round(float64(currentBalance*rates.Bid*100)) / 100)

	return convertedBalance, nil
}

func (*Service) ValidateCurrency(verifiedCurrency string) error {
	currenciesList := []string{eur, rub, usd}

	for _, v := range currenciesList {
		if verifiedCurrency == v {
			return nil
		}
	}

	return ErrCurrencyNotValid
}
