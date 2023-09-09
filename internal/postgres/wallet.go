package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	walletmodel "github.com/AlexZav1327/service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	createWalletQuery = `
	INSERT INTO wallet (wallet_id, owner, balance, currency) 
	VALUES ($1, $2, $3, $4)
	RETURNING wallet_id, owner, balance, currency, created_at, updated_at;
`
	getWalletsListQuery = `
	SELECT wallet_id, owner, balance, currency, created_at, updated_at 
	FROM wallet;
`
	getWalletQuery = `
	SELECT wallet_id, owner, balance, currency, created_at, updated_at 
	FROM wallet
	WHERE wallet_id = $1;
`
	updateWalletQuery = `
	UPDATE wallet 
	SET owner = $2, balance = $3, updated_at = $4
	WHERE wallet_id = $1
	RETURNING wallet_id, owner, balance, currency, created_at, updated_at;
`
	deleteWalletQuery = `
	DELETE FROM wallet 
	WHERE wallet_id = $1;
`
)

var (
	ErrWalletNotFound  = errors.New("no such wallet")
	ErrWalletsNotFound = errors.New("no wallets")
)

func (p *Postgres) CreateWallet(ctx context.Context, id uuid.UUID, owner string, balance float32, currency string) (walletmodel.WalletInstance, error) { //nolint:lll
	row := p.db.QueryRow(ctx, createWalletQuery, id, owner, balance, currency)

	var wallet walletmodel.WalletInstance

	err := row.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Balance, &wallet.Currency, &wallet.Created, &wallet.Updated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return walletmodel.WalletInstance{}, ErrWalletNotFound
		}

		return walletmodel.WalletInstance{}, fmt.Errorf("row.Scan: %w", err)
	}

	return wallet, nil
}

func (p *Postgres) GetWalletsList(ctx context.Context) ([]walletmodel.WalletInstance, error) {
	rows, err := p.db.Query(ctx, getWalletsListQuery)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}

	defer rows.Close()

	var walletsList []walletmodel.WalletInstance

	for rows.Next() {
		var wallet walletmodel.WalletInstance

		err := rows.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Balance, &wallet.Currency, &wallet.Created, &wallet.Updated)
		if err != nil {
			return nil, fmt.Errorf("row.Scan: %w", err)
		}

		walletsList = append(walletsList, wallet)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	if len(walletsList) == 0 {
		return nil, ErrWalletsNotFound
	}

	return walletsList, nil
}

func (p *Postgres) GetWallet(ctx context.Context, id string) (walletmodel.WalletInstance, error) {
	row := p.db.QueryRow(ctx, getWalletQuery, id)

	var wallet walletmodel.WalletInstance

	err := row.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Balance, &wallet.Currency, &wallet.Created, &wallet.Updated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return walletmodel.WalletInstance{}, ErrWalletNotFound
		}

		return walletmodel.WalletInstance{}, fmt.Errorf("row.Scan: %w", err)
	}

	return wallet, nil
}

func (p *Postgres) UpdateWallet(ctx context.Context, id string, owner string, balance float32) (walletmodel.WalletInstance, error) { //nolint:lll
	row := p.db.QueryRow(ctx, updateWalletQuery, id, owner, balance, time.Now())

	var wallet walletmodel.WalletInstance

	err := row.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Balance, &wallet.Currency, &wallet.Created, &wallet.Updated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return walletmodel.WalletInstance{}, ErrWalletNotFound
		}

		return walletmodel.WalletInstance{}, fmt.Errorf("row.Scan: %w", err)
	}

	return wallet, nil
}

func (p *Postgres) DeleteWallet(ctx context.Context, id string) error {
	commandTag, err := p.db.Exec(ctx, deleteWalletQuery, id)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	if commandTag.RowsAffected() != 1 {
		return ErrWalletNotFound
	}

	return nil
}
