package walletservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/AlexZav1327/service/models"
	"github.com/sirupsen/logrus"
)

const (
	eur = "EUR"
	rub = "RUB"
	usd = "USD"
)

var (
	ErrOverdraft       = errors.New("overdrafts are not allowed")
	ErrInvalidCurrency = errors.New("currency is invalid")
)

type Service struct {
	pg  WalletStore
	log *logrus.Entry
}

type WalletStore interface {
	CreateWallet(ctx context.Context, wallet models.RequestWalletInstance) (models.ResponseWalletInstance, error)
	GetWalletsList(ctx context.Context) ([]models.ResponseWalletInstance, error)
	GetWallet(ctx context.Context, id string) (models.ResponseWalletInstance, error)
	UpdateWallet(ctx context.Context, wallet models.RequestWalletInstance) (models.ResponseWalletInstance, error)
	DeleteWallet(ctx context.Context, id string) error
	ManageBalance(ctx context.Context, id string, balance float32) (models.ResponseWalletInstance, error)
	TransferFunds(ctx context.Context, idSrc, idDst string, balanceSrc, balanceDst float32) (
		models.ResponseWalletInstance, error)
	Idempotency(ctx context.Context, id string) error
}

func New(pg WalletStore, log *logrus.Logger) *Service {
	return &Service{
		pg:  pg,
		log: log.WithField("module", "service"),
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet models.RequestWalletInstance) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(wallet.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrInvalidCurrency
	}

	createdWallet, err := s.pg.CreateWallet(ctx, wallet)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.CreateWallet: %w", err)
	}

	return createdWallet, nil
}

func (s *Service) GetWalletsList(ctx context.Context) ([]models.ResponseWalletInstance, error) {
	walletsList, err := s.pg.GetWalletsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("pg.GetWalletsList: %w", err)
	}

	return walletsList, nil
}

func (s *Service) GetWallet(ctx context.Context, id string) (models.ResponseWalletInstance, error) {
	wallet, err := s.pg.GetWallet(ctx, id)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.GetWallet: %w", err)
	}

	return wallet, nil
}

func (s *Service) UpdateWallet(ctx context.Context, wallet models.RequestWalletInstance) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(wallet.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrInvalidCurrency
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

	updatedWallet, err := s.pg.UpdateWallet(ctx, wallet)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.UpdateWallet: %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) DeleteWallet(ctx context.Context, id string) error {
	err := s.pg.DeleteWallet(ctx, id)
	if err != nil {
		return fmt.Errorf("pg.DeleteWallet: %w", err)
	}

	return nil
}

func (s *Service) DepositFunds(ctx context.Context, id string, depositFunds models.FundsOperations) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(depositFunds.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrInvalidCurrency
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

	updatedWallet, err := s.pg.ManageBalance(ctx, id, balance)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.ManageFunds: %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) WithdrawFunds(ctx context.Context, id string, withdrawFunds models.FundsOperations) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(withdrawFunds.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrInvalidCurrency
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

	updatedWallet, err := s.pg.ManageBalance(ctx, id, balance)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.ManageFunds: %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) TransferFunds(ctx context.Context, idSrc, idDst string, transferFunds models.FundsOperations) (
	models.ResponseWalletInstance, error,
) {
	err := s.ValidateCurrency(transferFunds.Currency)
	if err != nil {
		return models.ResponseWalletInstance{}, ErrInvalidCurrency
	}

	currentSrcWallet, err := s.pg.GetWallet(ctx, idSrc)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.GetWallet: %w", err)
	}

	balanceSrc := currentSrcWallet.Balance - transferFunds.Amount

	if transferFunds.Currency != currentSrcWallet.Currency {
		convertedWithdrawAmount, err := s.ConvertCurrency(
			ctx,
			transferFunds.Currency,
			currentSrcWallet.Currency,
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
		convertedDepositAmount, err := s.ConvertCurrency(
			ctx,
			transferFunds.Currency,
			currentDstWallet.Currency,
			transferFunds.Amount,
		)
		if err != nil {
			return models.ResponseWalletInstance{}, fmt.Errorf("ConvertCurrency: %w", err)
		}

		balanceDst = currentDstWallet.Balance + convertedDepositAmount
	}

	updatedWallet, err := s.pg.TransferFunds(ctx, idSrc, idDst, balanceSrc, balanceDst)
	if err != nil {
		return models.ResponseWalletInstance{}, fmt.Errorf("pg.TransferFunds: %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) ConvertCurrency(ctx context.Context, currentCurrency, requestedCurrency string, currentBalance float32) (float32, error) { //nolint:lll
	endpoint := fmt.Sprintf("http://localhost:8091/api/v1/xr?from=%s&to=%s", currentCurrency, requestedCurrency)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("http.NewRequest: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("http.DefaultClient.Do: %w", err)
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			s.log.Warningf("resp.Body.Close: %s", err)
		}
	}()

	var rates models.ExchangeRate

	err = json.NewDecoder(response.Body).Decode(&rates)
	if err != nil {
		return 0, fmt.Errorf("json.NewDecoder.Decode: %w", err)
	}

	convertedBalance := currentBalance * rates.Bid

	return convertedBalance, nil
}

func (s *Service) Idempotency(ctx context.Context, key string) error {
	err := s.pg.Idempotency(ctx, key)
	if err != nil {
		return fmt.Errorf("pg.CheckIdempotency: %w", err)
	}

	return nil
}

func (*Service) ValidateCurrency(verifiedCurrency string) error {
	currenciesList := []string{eur, rub, usd}

	for _, v := range currenciesList {
		if verifiedCurrency == v {
			return nil
		}
	}

	return ErrInvalidCurrency
}
