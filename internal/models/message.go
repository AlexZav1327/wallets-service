package models

type MessageTemplate struct {
	Receiver    string `json:"receiver"`
	Message     string `json:"message"`
	Attachments []any  `json:"attachments"`
}
