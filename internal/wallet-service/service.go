package walletservice

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/AlexZav1327/service/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	ErrWrongOwnerName       = errors.New("wallet owner name is not formatted correctly")
	ErrOverdraft            = errors.New("overdrafts are not allowed")
	ErrWrongCurrency        = errors.New("currency is not valid")
	ErrUnchangeableCurrency = errors.New("currency is unchangeable")
)

type Service struct {
	pg  WalletStore
	log *logrus.Entry
}

type WalletStore interface {
	CreateWallet(ctx context.Context, id uuid.UUID, owner string, balance float32, currency string) (models.WalletInstance, error) //nolint:lll
	GetWalletsList(ctx context.Context) ([]models.WalletInstance, error)
	GetWallet(ctx context.Context, id string) (models.WalletInstance, error)
	UpdateWallet(ctx context.Context, id string, owner string, balance float32) (models.WalletInstance, error)
	DeleteWallet(ctx context.Context, id string) error
}

func New(pg WalletStore, log *logrus.Logger) *Service {
	return &Service{
		pg:  pg,
		log: log.WithField("module", "service"),
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet models.WalletInstance) (models.WalletInstance, error) { //nolint:lll
	if utf8.RuneCountInString(wallet.Owner) < 3 || utf8.RuneCountInString(wallet.Owner) > 50 {
		return models.WalletInstance{}, ErrWrongOwnerName
	}

	if wallet.Balance < 0 {
		return models.WalletInstance{}, ErrOverdraft
	}

	if wallet.Currency != "RUB" && wallet.Currency != "USD" && wallet.Currency != "EUR" {
		return models.WalletInstance{}, ErrWrongCurrency
	}

	createdWallet, err := s.pg.CreateWallet(ctx, wallet.WalletID, wallet.Owner, wallet.Balance, wallet.Currency)
	if err != nil {
		return models.WalletInstance{}, fmt.Errorf("pg.CreateWallet: %w", err)
	}

	return createdWallet, nil
}

func (s *Service) GetWalletsList(ctx context.Context) ([]models.WalletInstance, error) {
	walletsList, err := s.pg.GetWalletsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("FetchWalletsList: %w", err)
	}

	return walletsList, nil
}

func (s *Service) GetWallet(ctx context.Context, id string) (models.WalletInstance, error) {
	wallet, err := s.pg.GetWallet(ctx, id)
	if err != nil {
		return models.WalletInstance{}, fmt.Errorf("pg.FetchWalletByID: %w", err)
	}

	return wallet, nil
}

func (s *Service) UpdateWallet(ctx context.Context, wallet models.WalletInstance) (models.WalletInstance, error) { //nolint:lll
	currentWallet, err := s.pg.GetWallet(ctx, wallet.WalletID.String())
	if err != nil {
		return models.WalletInstance{}, fmt.Errorf("pg.FetchWalletByID: %w", err)
	}

	if wallet.Owner == "" {
		wallet.Owner = currentWallet.Owner
	} else if utf8.RuneCountInString(wallet.Owner) < 3 || utf8.RuneCountInString(wallet.Owner) > 50 {
		return models.WalletInstance{}, ErrWrongOwnerName
	}

	if currentWallet.Balance+wallet.Balance < 0 {
		return models.WalletInstance{}, ErrOverdraft
	}

	if currentWallet.Currency != wallet.Currency && wallet.Currency != "" {
		return models.WalletInstance{}, ErrUnchangeableCurrency
	}

	id := wallet.WalletID.String()
	owner := wallet.Owner
	balance := currentWallet.Balance + wallet.Balance

	updatedWallet, err := s.pg.UpdateWallet(ctx, id, owner, balance)
	if err != nil {
		return models.WalletInstance{}, fmt.Errorf("pg.UpdateWallet: %w", err)
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
