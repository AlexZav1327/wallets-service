package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/AlexZav1327/service/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
)

//go:embed migrations
var migrations embed.FS

var ErrNoRecords = errors.New("no records")

type Postgres struct {
	db  *pgx.Conn
	log *logrus.Entry
	dsn string
}

func ConnectDB(ctx context.Context, log *logrus.Logger, dsn string) (*Postgres, error) {
	db, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx.Connect: %w", err)
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	return &Postgres{
		db:  db,
		log: log.WithField("module", "postgres"),
		dsn: dsn,
	}, nil
}

func (p *Postgres) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", p.dsn)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}

	defer func() {
		err := conn.Close()
		if err != nil {
			p.log.Warningf("conn.Close: %s", err)
		}
	}()

	assetDir := func() func(string) ([]string, error) {
		return func(path string) ([]string, error) {
			dirEntry, err := migrations.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("migrations.ReadDir: %w", err)
			}

			entries := make([]string, 0)

			for _, e := range dirEntry {
				entries = append(entries, e.Name())
			}

			return entries, nil
		}
	}()

	asset := migrate.AssetMigrationSource{
		Asset:    migrations.ReadFile,
		AssetDir: assetDir,
		Dir:      "migrations",
	}

	_, err = migrate.Exec(conn, "postgres", asset, direction)
	if err != nil {
		return fmt.Errorf("migrate.Exec: %w", err)
	}

	return nil
}

func (p *Postgres) CreateWallet(ctx context.Context, id uuid.UUID, owner string, balance float32) ([]models.WalletData, error) { //nolint:lll
	query := `
		INSERT INTO wallet (wallet_id, owner, balance) 
		VALUES ($1, $2, $3)
		RETURNING wallet_id, owner, balance, created_at, updated_at;
	`

	row := p.db.QueryRow(ctx, query, id, owner, balance)

	var wallet models.WalletData

	err := row.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Balance, &wallet.Created, &wallet.Updated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRecords
		}

		return nil, fmt.Errorf("row.Scan: %w", err)
	}

	return []models.WalletData{wallet}, nil
}

func (p *Postgres) FetchWalletsList(ctx context.Context) ([]models.WalletData, error) {
	query := `
		SELECT wallet_id, owner, balance, created_at, updated_at 
		FROM wallet;
	`

	rows, err := p.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}

	defer rows.Close()

	var walletsList []models.WalletData

	for rows.Next() {
		var wallet models.WalletData

		err := rows.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Balance, &wallet.Created, &wallet.Updated)
		if err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}

		walletsList = append(walletsList, wallet)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	return walletsList, nil
}

func (p *Postgres) FetchWalletByID(ctx context.Context, id string) ([]models.WalletData, error) {
	query := `
		SELECT wallet_id, owner, balance, created_at, updated_at 
		FROM wallet
		WHERE wallet_id = $1;
	`

	row := p.db.QueryRow(ctx, query, id)

	var wallet models.WalletData

	err := row.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Balance, &wallet.Created, &wallet.Updated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRecords
		}

		return nil, fmt.Errorf("row.Scan: %w", err)
	}

	return []models.WalletData{wallet}, nil
}

func (p *Postgres) UpdateWallet(ctx context.Context, id string, owner string, balance float32) ([]models.WalletData, error) { //nolint:lll
	query := `
		UPDATE wallet 
		SET owner = $2, balance = $3, updated_at = $4 
		WHERE wallet_id = $1
		RETURNING wallet_id, owner, balance, created_at, updated_at;
	`

	row := p.db.QueryRow(ctx, query, id, owner, balance, time.Now())

	var wallet models.WalletData

	err := row.Scan(&wallet.WalletID, &wallet.Owner, &wallet.Balance, &wallet.Created, &wallet.Updated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRecords
		}

		return nil, fmt.Errorf("row.Scan: %w", err)
	}

	return []models.WalletData{wallet}, nil
}

func (p *Postgres) DeleteWallet(ctx context.Context, id string) error {
	query := `
		DELETE FROM wallet 
		WHERE wallet_id = $1;
	`

	_, err := p.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}
