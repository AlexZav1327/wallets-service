package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	migrate "github.com/rubenv/sql-migrate"
	log "github.com/sirupsen/logrus"
)

//go:embed migrations
var migrations embed.FS

var ErrNoRows = errors.New("no records")

type Postgres struct {
	db  *pgx.Conn
	dsn string
}

func ConnectDB(ctx context.Context, dsn string) (*Postgres, error) {
	db, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx.Connect(ctx, dsn): %w", err)
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.Ping(ctx): %w", err)
	}

	return &Postgres{
		db:  db,
		dsn: dsn,
	}, nil
}

func (p *Postgres) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", p.dsn)
	if err != nil {
		return fmt.Errorf("sql.Open(\"pgx\", p.dsn): %w", err)
	}

	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("conn.Close(): %s", err)
		}
	}()

	assetDir := func() func(string) ([]string, error) {
		return func(path string) ([]string, error) {
			dirEntry, err := migrations.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("migrations.ReadDir(path): %w", err)
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
		return fmt.Errorf("migrate.Exec(conn, \"postgres\", asset, direction): %w", err)
	}

	return nil
}

func (p *Postgres) StoreAccessData(ctx context.Context, userIP string, accessTime string) error {
	query := `INSERT INTO access_data (user_ip, access_time) VALUES ($1, $2 AT TIME ZONE 'Europe/Moscow');`

	_, err := p.db.Exec(ctx, query, userIP, accessTime)
	if err != nil {
		return fmt.Errorf("p.db.Exec(ctx, query, userIP, accessTime): %w", err)
	}

	return nil
}

func (p *Postgres) FetchAccessData(ctx context.Context) (map[string][]time.Time, error) {
	query := `SELECT user_ip, access_time FROM access_data;`

	rows, err := p.db.Query(ctx, query)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("p.db.Query(ctx, query): %w", err)
	}

	defer rows.Close()

	data := make(map[string][]time.Time)

	ok := rows.Next()

	if !ok {
		return nil, fmt.Errorf("rows.Next(): %w", ErrNoRows)
	}

	for rows.Next() {
		var userIP string

		var accessTime time.Time

		err := rows.Scan(&userIP, &accessTime)
		if err != nil {
			log.Panicf("rows.Scan(&userIP, &accessTime): %s", err)
		}

		_, ok := data[userIP]

		if ok {
			data[userIP] = append(data[userIP], accessTime)
		} else {
			data[userIP] = []time.Time{accessTime}
		}

		err = rows.Err()
		if err != nil {
			log.Panicf("rows.Err(): %s", err)
		}
	}

	return data, nil
}
