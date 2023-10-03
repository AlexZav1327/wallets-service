package postgres

import (
	"context"
	"errors"
	"fmt"

	walletmodel "github.com/AlexZav1327/service/models"
	"github.com/jackc/pgx/v5"
)

const (
	createWalletQuery = `
	INSERT INTO wallet (wallet_id, owner, currency) 
	VALUES ($1, $2, $3)
	RETURNING wallet_id, owner, currency, balance, created_at, updated_at;
`
	getWalletsListQuery = `
	SELECT wallet_id, owner, currency, balance, created_at, updated_at 
	FROM wallet;
`
	getWalletQuery = `
	SELECT wallet_id, owner, currency, balance, created_at, updated_at 
	FROM wallet
	WHERE wallet_id = $1;
`
	getWalletHistoryQuery = `
	SELECT wallet_id, owner, currency, balance, created_at, operation_type
	FROM history
	WHERE wallet_id = $1 AND created_at >= $2 AND created_at <= $3;
`
	updateWalletQuery = `
	UPDATE wallet 
	SET owner = $2, currency = $3, balance = $4, updated_at = now()
	WHERE wallet_id = $1
	RETURNING wallet_id, owner, currency, balance, created_at, updated_at;
`
	deleteWalletQuery = `
	DELETE FROM wallet 
	WHERE wallet_id = $1;
`
	manageFundsQuery = `
	UPDATE wallet
	SET balance = $2, updated_at = now()
	WHERE wallet_id = $1
	RETURNING wallet_id, owner, currency, balance, created_at, updated_at;
`
	checkTransactionKeyQuery = `
	INSERT INTO idempotency (transaction_key) 
	VALUES ($1)
`
)

var (
	ErrWalletNotFound  = errors.New("no such wallet")
	ErrWalletsNotFound = errors.New("no wallets")
)

func (p *Postgres) CreateWallet(ctx context.Context, wallet walletmodel.RequestWalletInstance) (
	walletmodel.ResponseWalletInstance, error,
) {
	row := p.db.QueryRow(ctx, createWalletQuery, wallet.WalletID, wallet.Owner, wallet.Currency)

	var createdWallet walletmodel.ResponseWalletInstance

	err := row.Scan(
		&createdWallet.WalletID,
		&createdWallet.Owner,
		&createdWallet.Currency,
		&createdWallet.Balance,
		&createdWallet.Created,
		&createdWallet.Updated,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return walletmodel.ResponseWalletInstance{}, ErrWalletNotFound
		}

		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("row.Scan: %w", err)
	}

	return createdWallet, nil
}

func (p *Postgres) GetWalletsList(ctx context.Context) ([]walletmodel.ResponseWalletInstance, error) {
	rows, err := p.db.Query(ctx, getWalletsListQuery)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}

	defer rows.Close()

	var walletsList []walletmodel.ResponseWalletInstance

	for rows.Next() {
		var wallet walletmodel.ResponseWalletInstance

		err = rows.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Currency, &wallet.Balance, &wallet.Created, &wallet.Updated)
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

func (p *Postgres) GetWallet(ctx context.Context, id string) (walletmodel.ResponseWalletInstance, error) {
	row := p.db.QueryRow(ctx, getWalletQuery, id)

	var wallet walletmodel.ResponseWalletInstance

	err := row.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Currency, &wallet.Balance, &wallet.Created, &wallet.Updated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return walletmodel.ResponseWalletInstance{}, ErrWalletNotFound
		}

		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("row.Scan: %w", err)
	}

	return wallet, nil
}

func (p *Postgres) GetWalletHistory(ctx context.Context, walletHistoryPeriod walletmodel.RequestWalletHistory) (
	[]walletmodel.ResponseWalletHistory, error,
) {
	rows, err := p.db.Query(
		ctx,
		getWalletHistoryQuery,
		walletHistoryPeriod.WalletID,
		walletHistoryPeriod.PeriodStart,
		walletHistoryPeriod.PeriodEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}

	defer rows.Close()

	var walletHistory []walletmodel.ResponseWalletHistory

	for rows.Next() {
		var wallet walletmodel.ResponseWalletHistory

		err = rows.Scan(
			&wallet.WalletID,
			&wallet.Owner,
			&wallet.Currency,
			&wallet.Balance,
			&wallet.Created,
			&wallet.Operation,
		)
		if err != nil {
			return nil, fmt.Errorf("row.Scan: %w", err)
		}

		walletHistory = append(walletHistory, wallet)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	if len(walletHistory) == 0 {
		return nil, ErrWalletNotFound
	}

	return walletHistory, nil
}

func (p *Postgres) UpdateWallet(ctx context.Context, wallet walletmodel.RequestWalletInstance) (
	walletmodel.ResponseWalletInstance, error,
) {
	row := p.db.QueryRow(
		ctx,
		updateWalletQuery,
		wallet.WalletID,
		wallet.Owner,
		wallet.Currency,
		wallet.Balance,
	)

	var updatedWallet walletmodel.ResponseWalletInstance

	err := row.Scan(
		&updatedWallet.WalletID,
		&updatedWallet.Owner,
		&updatedWallet.Currency,
		&updatedWallet.Balance,
		&updatedWallet.Created,
		&updatedWallet.Updated,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return walletmodel.ResponseWalletInstance{}, ErrWalletNotFound
		}

		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("row.Scan: %w", err)
	}

	return updatedWallet, nil
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

func (p *Postgres) ManageBalance(ctx context.Context, id string, balance float32) (
	walletmodel.ResponseWalletInstance, error,
) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("db.BeginTx: %w", err)
	}

	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				p.log.Warningf("tx.Rollback: %s", err)
			}
		}
	}()

	updatedWallet, err := p.queryRowToWallet(ctx, manageFundsQuery, id, balance)

	err = tx.Commit(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return updatedWallet, nil
}

func (p *Postgres) TransferFunds(ctx context.Context, idSrc, idDst string, balanceSrc, balanceDst float32,
) (walletmodel.ResponseWalletInstance, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("db.BeginTx: %w", err)
	}

	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				p.log.Warningf("tx.Rollback: %s", err)
			}
		}
	}()

	_, err = p.queryRowToWallet(ctx, manageFundsQuery, idSrc, balanceSrc)

	dstWallet, err := p.queryRowToWallet(ctx, manageFundsQuery, idDst, balanceDst)

	err = tx.Commit(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return dstWallet, nil
}

func (p *Postgres) Idempotency(ctx context.Context, key string) error {
	row := p.db.QueryRow(ctx, checkTransactionKeyQuery, key)

	err := row.Scan()
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("row.Scan: %w", err)
	}

	return nil
}

func (p *Postgres) queryRowToWallet(ctx context.Context, query, id string, balance float32) (
	walletmodel.ResponseWalletInstance, error,
) {
	row := p.db.QueryRow(ctx, query, id, balance)

	var wallet walletmodel.ResponseWalletInstance

	err := row.Scan(
		&wallet.WalletID,
		&wallet.Owner,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.Created,
		&wallet.Updated,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return walletmodel.ResponseWalletInstance{}, ErrWalletNotFound
		}

		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("rowSrc.Scan: %w", err)
	}

	return wallet, nil
}
