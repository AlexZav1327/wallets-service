package notifications

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Notifications struct {
	log *logrus.Entry
}

func New(log *logrus.Logger) *Notifications {
	return &Notifications{
		log: log.WithField("module", "notifications"),
	}
}

func (n *Notifications) Notify(_ context.Context, _ []byte) error {
	return nil
}
