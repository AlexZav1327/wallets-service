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
		return nil, fmt.Errorf("pgx.Connect(ctx, dsn): %w", err)
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.Ping(ctx): %w", err)
	}

	return &Postgres{
		db:  db,
		log: log.WithField("module", "db"),
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
			p.log.Warningf("conn.Close(): %s", err)
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

func (p *Postgres) StoreAccessData(ctx context.Context, userIP string) error {
	query := `INSERT INTO access_data (user_ip) VALUES ($1);`

	_, err := p.db.Exec(ctx, query, userIP)
	if err != nil {
		return fmt.Errorf("db.Exec(ctx, query, userIP, accessTime): %w", err)
	}

	return nil
}

func (p *Postgres) FetchAccessData(ctx context.Context) (map[string][]time.Time, error) {
	query := `SELECT user_ip, access_time FROM access_data;`

	rows, err := p.db.Query(ctx, query)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("db.Query(ctx, query): %w", err)
	}

	defer rows.Close()

	data := make(map[string][]time.Time)

	for rows.Next() {
		var userIP string

		var accessTime time.Time

		err := rows.Scan(&userIP, &accessTime)
		if err != nil {
			p.log.Warningf("rows.Scan(&userIP, &accessTime): %s", err)
		}

		_, ok := data[userIP]

		if ok {
			data[userIP] = append(data[userIP], accessTime)
		} else {
			data[userIP] = []time.Time{accessTime}
		}
	}

	err = rows.Err()
	if err != nil {
		p.log.Warningf("rows.Err(): %s", err)
	}

	if len(data) == 0 {
		return nil, ErrNoRecords
	}

	return data, nil
}
