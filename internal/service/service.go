package service

import (
	"context"
	"fmt"

	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/sirupsen/logrus"
)

const dataTimeFmt = "2006-01-02 15:04:05"

type AccessData struct {
	pg  postgres.Postgres
	log *logrus.Entry
}

func NewAccessData(pg *postgres.Postgres, log *logrus.Logger) *AccessData {
	return &AccessData{
		pg:  *pg,
		log: log.WithField("module", "service"),
	}
}

func (a *AccessData) SaveAccessData(ctx context.Context, ip string) error {
	err := a.pg.StoreAccessData(ctx, ip)
	if err != nil {
		return fmt.Errorf("StoreAccessData(ip, time): %w", err)
	}

	return nil
}

func (*AccessData) ShowCurrentAccessData(ip string, time string) string {
	return fmt.Sprintf("Your IP address is: %s\tCurrent date and time is: %s", ip, time)
}

func (a *AccessData) ShowPreviousAccessData(ctx context.Context) (string, error) {
	data, err := a.pg.FetchAccessData(ctx)
	if err != nil {
		return "", fmt.Errorf("FetchAccessData(): %w", err)
	}

	var convertedData string

	for ip, time := range data {
		var convertedTime string
		for _, v := range time {
			convertedTime += fmt.Sprintf("%s; ", v.Format(dataTimeFmt))
		}

		convertedData += fmt.Sprintf("IP address: %s\tDate and time: %s\n", ip, convertedTime[:len(convertedTime)-2])
	}

	return convertedData, nil
}
