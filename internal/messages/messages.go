package messages

import (
	"encoding/json"
	"fmt"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/sirupsen/logrus"
)

type Message struct {
	log *logrus.Entry
}

func New(log *logrus.Logger) *Message {
	return &Message{
		log: log.WithField("module", "messages"),
	}
}

func (n *Message) CreateMessage(wallet models.ResponseWalletInstance) ([]byte, error) {
	message := models.MessageTemplate{
		Receiver:    wallet.Email,
		Message:     "",
		Attachments: nil,
	}

	bytes, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	return bytes, nil
}
