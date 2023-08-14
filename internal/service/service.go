package service

import (
	"context"
	"fmt"

	"github.com/AlexZav1327/service/internal/postgres"
	log "github.com/sirupsen/logrus"
)

type AccessData struct {
	pg *postgres.Postgres
}

func NewAccessData(pg *postgres.Postgres) *AccessData {
	return &AccessData{
		pg: pg,
	}
}

func (a *AccessData) SaveAccessData(ctx context.Context, ip string, time string) error {
	err := a.pg.StoreAccessData(ctx, ip, time)
	if err != nil {
		return fmt.Errorf("pg.StoreAccessData(ip, time): %w", err)
	}

	return nil
}

func (*AccessData) ShowCurrentAccessData(ip string, time string) string {
	return fmt.Sprintf("Your IP address is: %s\tCurrent date and time is: %s", ip, time)
}

func (a *AccessData) ShowPreviousAccessData(ctx context.Context) string {
	data, err := a.pg.FetchAccessData(ctx)
	if err != nil {
		log.Panicf("a.pg.FetchAccessData(): %s", err)
	}

	var convertedData string

	for ip, time := range data {
		var convertedTime string
		for _, v := range time {
			convertedTime += fmt.Sprintf("%s; ", v.Format("2006-01-02 15:04:05"))
		}

		convertedData += fmt.Sprintf("IP address: %s\tDate and time: %s\n", ip, convertedTime[:len(convertedTime)-2])
	}

	return convertedData
}
