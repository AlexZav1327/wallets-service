package service

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

type Wallet struct {
	pg  WalletStore
	log *logrus.Entry
}

type WalletStore interface {
	CreateWallet(ctx context.Context, id uuid.UUID, owner string, balance float32, currency string) (models.WalletInstance, error) //nolint:lll
	FetchWalletsList(ctx context.Context) ([]models.WalletInstance, error)
	FetchWalletByID(ctx context.Context, id string) (models.WalletInstance, error)
	UpdateWallet(ctx context.Context, id string, owner string, balance float32) (models.WalletInstance, error)
	DeleteWallet(ctx context.Context, id string) error
}

func NewWallet(pg WalletStore, log *logrus.Logger) *Wallet {
	return &Wallet{
		pg:  pg,
		log: log.WithField("module", "service"),
	}
}

func (w *Wallet) Create(ctx context.Context, wallet models.WalletInstance) (models.WalletInstance, error) {
	if utf8.RuneCountInString(wallet.Owner) < 3 || utf8.RuneCountInString(wallet.Owner) > 50 {
		return models.WalletInstance{}, ErrWrongOwnerName
	}

	if wallet.Balance < 0 {
		return models.WalletInstance{}, ErrOverdraft
	}

	if wallet.Currency != "RUB" && wallet.Currency != "USD" && wallet.Currency != "EUR" {
		return models.WalletInstance{}, ErrWrongCurrency
	}

	createdWallet, err := w.pg.CreateWallet(ctx, wallet.WalletID, wallet.Owner, wallet.Balance, wallet.Currency)
	if err != nil {
		return models.WalletInstance{}, fmt.Errorf("pg.CreateWallet: %w", err)
	}

	return createdWallet, nil
}

func (w *Wallet) GetList(ctx context.Context) ([]models.WalletInstance, error) {
	walletsList, err := w.pg.FetchWalletsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("FetchWalletsList: %w", err)
	}

	return walletsList, nil
}

func (w *Wallet) Get(ctx context.Context, id string) (models.WalletInstance, error) {
	wallet, err := w.pg.FetchWalletByID(ctx, id)
	if err != nil {
		return models.WalletInstance{}, fmt.Errorf("pg.FetchWalletByID: %w", err)
	}

	return wallet, nil
}

func (w *Wallet) Update(ctx context.Context, wallet models.WalletInstance) (models.WalletInstance, error) {
	currentWallet, err := w.pg.FetchWalletByID(ctx, wallet.WalletID.String())
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

	updatedWallet, err := w.pg.UpdateWallet(ctx, id, owner, balance)
	if err != nil {
		return models.WalletInstance{}, fmt.Errorf("pg.UpdateWallet: %w", err)
	}

	return updatedWallet, nil
}

func (w *Wallet) Delete(ctx context.Context, id string) error {
	err := w.pg.DeleteWallet(ctx, id)
	if err != nil {
		return fmt.Errorf("pg.DeleteWallet: %w", err)
	}

	return nil
}
