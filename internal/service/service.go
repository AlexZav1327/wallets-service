package service

import (
	"context"
	"fmt"

	"github.com/AlexZav1327/service/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Wallet struct {
	pg  WalletStore
	log *logrus.Entry
}

type WalletStore interface {
	CreateWallet(ctx context.Context, id uuid.UUID, owner string, balance float32) ([]models.WalletData, error)
	FetchWalletsList(ctx context.Context) ([]models.WalletData, error)
	FetchWalletByID(ctx context.Context, id string) ([]models.WalletData, error)
	UpdateWallet(ctx context.Context, id string, owner string, balance float32) ([]models.WalletData, error)
	DeleteWallet(ctx context.Context, id string) error
}

func NewWallet(pg WalletStore, log *logrus.Logger) *Wallet {
	return &Wallet{
		pg:  pg,
		log: log.WithField("module", "service"),
	}
}

func (*Wallet) generateUUID() uuid.UUID {
	return uuid.New()
}

func (w *Wallet) Create(ctx context.Context, wallet models.WalletData) ([]models.WalletData, error) {
	id := w.generateUUID()

	createdWallet, err := w.pg.CreateWallet(ctx, id, *wallet.Owner, *wallet.Balance)
	if err != nil {
		return nil, fmt.Errorf("pg.CreateWallet: %w", err)
	}

	return createdWallet, nil
}

func (w *Wallet) GetList(ctx context.Context) ([]models.WalletData, error) {
	walletsList, err := w.pg.FetchWalletsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("FetchWalletsList: %w", err)
	}

	return walletsList, nil
}

func (w *Wallet) Get(ctx context.Context, id string) ([]models.WalletData, error) {
	wallet, err := w.pg.FetchWalletByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("pg.FetchWalletByID: %w", err)
	}

	return wallet, nil
}

func (w *Wallet) Update(ctx context.Context, id string, wallet models.WalletData) ([]models.WalletData, error) {
	data, err := w.pg.FetchWalletByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("pg.FetchWalletByID: %w", err)
	}

	currentWallet := data[0]
	newWallet := models.WalletData{}

	if wallet.Owner != nil && wallet.Owner != currentWallet.Owner {
		newWallet.Owner = wallet.Owner
	} else {
		newWallet.Owner = currentWallet.Owner
	}

	if wallet.Balance != nil && wallet.Balance != currentWallet.Balance {
		newWallet.Balance = wallet.Balance
	} else {
		newWallet.Balance = currentWallet.Balance
	}

	var updatedWallet []models.WalletData

	updatedWallet, err = w.pg.UpdateWallet(ctx, id, *newWallet.Owner, *newWallet.Balance)
	if err != nil {
		return nil, fmt.Errorf("pg.UpdateWallet: %w", err)
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
