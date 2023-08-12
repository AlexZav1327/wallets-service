package postgres

import (
	"context"
	"database/sql"
	"embed"
	"time"

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
		log.WithFields(log.Fields{
			"package":  "postgres",
			"function": "ConnectDB",
			"error":    err,
		}).Error("Unable to parse config")

		return nil, err
	}

	db, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "postgres",
			"function": "ConnectDB",
			"error":    err,
		}).Error("Unable to connect config")

		return nil, err
	}

	if err = db.Ping(ctx); err != nil {
		log.WithFields(log.Fields{
			"package":  "postgres",
			"function": "ConnectDB",
			"error":    err,
		}).Warning("Unable to ping")

		return nil, err
	}

	return &Postgres{
		db:  db,
		dsn: dsn,
	}, nil
}

func (p *Postgres) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", p.dsn)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "postgres",
			"function": "Migrate",
			"error":    err,
		}).Error("Unable to migrate")
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.WithFields(log.Fields{
				"package":  "postgres",
				"function": "Migrate",
				"error":    err,
			}).Warning("Unable to close migration connection")
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
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "postgres",
			"function": "Migrate",
			"error":    err,
		}).Error("Unable to execute a set of migrations")
	}

	return err
}

func (p *Postgres) StoreAccessData(userIP string, accessTime string) error {
	query := `INSERT INTO access_data (user_ip, access_time) VALUES ($1, $2);`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	_, err := p.db.Exec(ctx, query, userIP, accessTime)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "postgres",
			"function": "StoreAccessData",
			"error":    err,
		}).Error("Unable to insert access data to database")
	}

	return err
}

func (p *Postgres) FetchAccessData() (map[string][]string, error) {
	query := `SELECT user_ip, access_time FROM access_data;`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	rows, err := p.db.Query(ctx, query)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "postgres",
			"function": "FetchAccessData",
			"error":    err,
		}).Error("Unable to query access data from database")
	}

	defer rows.Close()

	data := make(map[string][]string)

	for rows.Next() {
		var userIP string

		var accessTime string

		if err := rows.Scan(&userIP, &accessTime); err != nil {
			log.WithFields(log.Fields{
				"package":  "postgres",
				"function": "FetchAccessData",
				"error":    err,
			}).Error("Unable to scan row")
		}

		_, ok := data[userIP]

		if ok {
			data[userIP] = append(data[userIP], accessTime)
		} else {
			data[userIP] = []string{accessTime}
		}

		if err = rows.Err(); err != nil {
			log.WithFields(log.Fields{
				"package":  "postgres",
				"function": "FetchAccessData",
				"error":    err,
			}).Error("Unable to iterate through access data rows")
		}
	}

	return data, err
}
