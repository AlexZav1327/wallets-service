package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	service WalletService
	log     *logrus.Entry
}

func NewHandler(service WalletService, log *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log.WithField("module", "handler"),
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var wallet models.WalletInstance

	wallet.WalletID = uuid.New()

	err := json.NewDecoder(r.Body).Decode(&wallet)
	if err != nil {
		h.log.Warningf("json.NewDecoder.Decode: %s", err)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	var createdWallet models.WalletInstance

	createdWallet, err = h.service.Create(r.Context(), wallet)
	if err != nil {
		h.log.Warningf("service.Create: %s", err)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(createdWallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) getList(w http.ResponseWriter, r *http.Request) {
	walletsList, err := h.service.GetList(r.Context())
	if err != nil {
		h.log.Warningf("service.GetList: %s", err)
		w.WriteHeader(http.StatusNotFound)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(walletsList)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")
	id := pathParts[3]

	wallet, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.log.Warningf("service.Get: %s", err)
		w.WriteHeader(http.StatusNotFound)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(wallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var wallet models.WalletInstance

	err := json.NewDecoder(r.Body).Decode(&wallet)
	if err != nil {
		h.log.Warningf("json.NewDecoder.Decode: %s", err)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	path := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")
	id := pathParts[3]

	wallet.WalletID = uuid.MustParse(id)

	var updatedWallet models.WalletInstance

	updatedWallet, err = h.service.Update(r.Context(), wallet)
	if err != nil && errors.Is(err, postgres.ErrWalletNotFound) {
		h.log.Warningf("service.Update: %s", err)
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err != nil && !errors.Is(err, postgres.ErrWalletNotFound) {
		h.log.Warningf("service.Update: %s", err)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(updatedWallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")
	id := pathParts[3]

	err := h.service.Delete(r.Context(), id)

	switch {
	case err != nil:
		h.log.Warningf("service.Delete: %s", err)
		w.WriteHeader(http.StatusNotFound)

		return

	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
