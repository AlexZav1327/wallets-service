package server

import (
	"net/http"
	"time"

	"github.com/AlexZav1327/service/internal/service"
	"github.com/sirupsen/logrus"
)

const dataTimeFmt = "2006-01-02 15:04:05"

type Handler struct {
	service service.AccessData
	log     *logrus.Entry
}

func NewHandler(service *service.AccessData, log *logrus.Logger) *Handler {
	return &Handler{
		service: *service,
		log:     log.WithField("module", "handler"),
	}
}

func (h *Handler) current(w http.ResponseWriter, r *http.Request) {
	err := h.service.SaveAccessData(r.Context(), r.RemoteAddr)
	if err != nil {
		h.log.Warningf("service.SaveAccessData(r.Context(), r.RemoteAddr): %s", err)
	}

	_, err = w.Write([]byte(h.service.ShowCurrentAccessData(r.RemoteAddr, h.getCurrentTime())))
	if err != nil {
		h.log.Warningf("service.ShowCurrentAccessData(r.RemoteAddr, getCurrentTime()): %s", err)
	}
}

func (h *Handler) previous(w http.ResponseWriter, r *http.Request) {
	previousAccessData, err := h.service.ShowPreviousAccessData(r.Context())
	if err != nil {
		h.log.Warningf("service.ShowPreviousAccessData(r.Context()): %s", err)
	}

	_, err = w.Write([]byte(previousAccessData))
	if err != nil {
		h.log.Warningf("Write([]byte(previousAccessData)): %s", err)
	}
}

func (h *Handler) getCurrentTime() string {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		h.log.Warningf("time.LoadLocation(): %s", err)
	}

	now := time.Now().In(location)

	return now.Format(dataTimeFmt)
}
