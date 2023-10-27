package postgres

import (
	"context"
	"errors"
	"fmt"

	walletmodel "github.com/AlexZav1327/service/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	createWalletQuery = `
	INSERT INTO wallet (wallet_id, owner, currency) 
	VALUES ($1, $2, $3)
	RETURNING wallet_id, owner, currency, balance, created_at, updated_at;
	`
	getWalletQuery = `
	SELECT wallet_id, owner, currency, balance, created_at, updated_at 
	FROM wallet
	WHERE wallet_id = $1;
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
	verifyTransactKeyQuery = `
	INSERT INTO idempotency (transaction_key)
	VALUES ($1)
	`
	walletID      = "wallet_id"
	owner         = "owner"
	currency      = "currency"
	balance       = "balance"
	createdAt     = "created_at"
	updatedAt     = "updated_at"
	operationType = "operation_type"
)

var (
	ErrWalletNotFound       = errors.New("no such wallet")
	ErrRequestNotIdempotent = errors.New("non-idempotent request")
	ErrNoWalletToDelete     = errors.New("no wallet found to delete")
)

type querier interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

func (p *Postgres) CreateWallet(ctx context.Context, wallet walletmodel.RequestWalletInstance) (
	walletmodel.ResponseWalletInstance, error,
) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("db.Begin: %w", err)
	}

	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				p.log.Warningf("tx.Rollback: %s", err)
			}
		}
	}()

	err = p.idempotency(ctx, tx, wallet.TransactionKey.String())
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("idempotency: %w", err)
	}

	row := tx.QueryRow(ctx, createWalletQuery, wallet.WalletID, wallet.Owner, wallet.Currency)

	var createdWallet walletmodel.ResponseWalletInstance

	err = row.Scan(
		&createdWallet.WalletID,
		&createdWallet.Owner,
		&createdWallet.Currency,
		&createdWallet.Balance,
		&createdWallet.Created,
		&createdWallet.Updated,
	)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("row.Scan: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return createdWallet, nil
}

func (p *Postgres) GetWalletsList(ctx context.Context, params walletmodel.ListingQueryParams) (
	[]walletmodel.ResponseWalletInstance, error,
) {
	tableColumnsList := map[string]string{
		walletID:  walletID,
		owner:     owner,
		currency:  currency,
		balance:   balance,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}

	var args []interface{}

	query := `
	SELECT *
	FROM wallet
	WHERE TRUE`

	updatedQuery, updatedArgs := p.buildQueryAndArgs(tableColumnsList, args, query, params)

	rows, err := p.db.Query(ctx, updatedQuery, updatedArgs...)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}

	defer rows.Close()

	walletsList := make([]walletmodel.ResponseWalletInstance, 0)

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

func (p *Postgres) GetWalletHistory(ctx context.Context, id string, params walletmodel.RequestWalletHistory) (
	[]walletmodel.ResponseWalletHistory, error,
) {
	tableColumnsList := map[string]string{
		walletID:      walletID,
		owner:         owner,
		currency:      currency,
		balance:       balance,
		createdAt:     createdAt,
		operationType: operationType,
	}

	var args []interface{}

	query := `
	SELECT *
	FROM history
	WHERE TRUE`

	args = append(args, id)
	query += fmt.Sprintf(` AND (wallet_id=$%d`, len(args))
	args = append(args, params.PeriodStart)
	query += fmt.Sprintf(` AND created_at >= $%d`, len(args))
	args = append(args, params.PeriodEnd)
	query += fmt.Sprintf(` AND created_at <= $%d)`, len(args))

	updatedQuery, updatedArgs := p.buildQueryAndArgs(tableColumnsList, args, query, params.ListingQueryParams)

	rows, err := p.db.Query(ctx, updatedQuery, updatedArgs...)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}

	defer rows.Close()

	walletHistory := make([]walletmodel.ResponseWalletHistory, 0)

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
		return ErrNoWalletToDelete
	}

	return nil
}

func (p *Postgres) ManageBalance(ctx context.Context, key uuid.UUID, id string, balance float32) (
	walletmodel.ResponseWalletInstance, error,
) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("db.Begin: %w", err)
	}

	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				p.log.Warningf("tx.Rollback: %s", err)
			}
		}
	}()

	err = p.idempotency(ctx, tx, key.String())
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("idempotency: %w", err)
	}

	updatedWallet, err := p.queryRowToWallet(ctx, tx, manageFundsQuery, id, balance)

	err = tx.Commit(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return updatedWallet, nil
}

func (p *Postgres) TransferFunds(ctx context.Context, key uuid.UUID, idSrc, idDst string, balanceSrc,
	balanceDst float32,
) (walletmodel.ResponseWalletInstance, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("db.Begin: %w", err)
	}

	defer func() {
		if err != nil {
			err = tx.Rollback(ctx)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				p.log.Warningf("tx.Rollback: %s", err)
			}
		}
	}()

	err = p.idempotency(ctx, tx, key.String())
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("idempotency: %w", err)
	}

	_, err = p.queryRowToWallet(ctx, tx, manageFundsQuery, idSrc, balanceSrc)

	dstWallet, err := p.queryRowToWallet(ctx, tx, manageFundsQuery, idDst, balanceDst)

	err = tx.Commit(ctx)
	if err != nil {
		return walletmodel.ResponseWalletInstance{}, fmt.Errorf("tx.Commit: %w", err)
	}

	return dstWallet, nil
}

func (p *Postgres) idempotency(ctx context.Context, q querier, key string) error {
	_, err := q.Exec(ctx, verifyTransactKeyQuery, key)

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgerrcode.UniqueViolation == pgErr.SQLState() {
			return ErrRequestNotIdempotent
		}
	}

	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

func (p *Postgres) queryRowToWallet(ctx context.Context, tx pgx.Tx, query, id string, balance float32) (
	walletmodel.ResponseWalletInstance, error,
) {
	row := tx.QueryRow(ctx, query, id, balance)

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

func (*Postgres) buildQueryAndArgs(tableColumnsList map[string]string, args []interface{}, query string,
	params walletmodel.ListingQueryParams,
) (string, []interface{}) {
	if params.TextFilter != "" {
		args = append(args, "%"+params.TextFilter+"%")
		query += fmt.Sprintf(` AND (owner ILIKE $%d OR currency ILIKE $%d)`, len(args), len(args))
	}

	order := ` ORDER BY created_at`

	sorting, ok := tableColumnsList[params.Sorting]
	if ok {
		order = fmt.Sprintf(` ORDER BY %s`, sorting)
	}

	if params.Descending {
		order += ` DESC`
	}

	query += order

	args = append(args, params.ItemsPerPage)
	query += fmt.Sprintf(` LIMIT $%d`, len(args))
	args = append(args, params.Offset)
	query += fmt.Sprintf(` OFFSET $%d`, len(args))

	return query, args
}
