package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
)

const resetTableQuery = `
	TRUNCATE TABLE wallet
`

//go:embed migrations
var migrations embed.FS

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

func (p *Postgres) ResetTable(ctx context.Context) error {
	_, err := p.db.Exec(ctx, resetTableQuery)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}
