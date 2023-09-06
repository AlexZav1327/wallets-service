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
	CreateWallet(ctx context.Context, id uuid.UUID, owner string, balance float32) (models.WalletInstance, error)
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
	createdWallet, err := w.pg.CreateWallet(ctx, wallet.WalletID, wallet.Owner, wallet.Balance)
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
	id := wallet.WalletID.String()
	owner := wallet.Owner
	balance := wallet.Balance

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
