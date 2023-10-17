package xrserver

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

type Handler struct {
	service RateService
	log     *logrus.Entry
}

func NewHandler(service RateService, log *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log.WithField("module", "xr_handler"),
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	currentRate, err := h.service.GetCurrentRate(from, to)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(currentRate)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}
