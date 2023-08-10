package postgres

import (
	"context"
	"database/sql"
	"embed"

	"github.com/jackc/pgx/v5"
	migrate "github.com/rubenv/sql-migrate"
	log "github.com/sirupsen/logrus"
)

//go:embed migrations
var migrations embed.FS

type Postgres struct {
	db  *pgx.Conn
	dsn string
}

func ConnectDB(ctx context.Context, dsn string) (*Postgres, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Infof("Parse config error: %s", err)
		return nil, err
	}

	db, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		log.Infof("Connect config error: %s", err)
		return nil, err
	}

	if err = db.Ping(ctx); err != nil {
		log.Infof("Ping error: %s", err)
		return nil, err
	}

	return &Postgres{
		db:  db,
		dsn: dsn,
	}, nil
}

func (p *Postgres) DC(ctx context.Context) {
	err := p.db.Close(ctx)
	if err != nil {
		return
	}
}

func (p *Postgres) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", p.dsn)
	if err != nil {
		return err
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Infof("err closing migration connection: %s", err)
		}
	}()

	assetDir := func() func(string) ([]string, error) {
		return func(path string) ([]string, error) {
			dirEntry, err := migrations.ReadDir(path)
			if err != nil {
				return nil, err
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
	return err
}

//
//func (p *Postgres) GetData(ctx context.Context, ip string)
