package walletserver

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/AlexZav1327/service/internal/postgres"
	walletservice "github.com/AlexZav1327/service/internal/wallet-service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultLimit     = 20
	defaultTimeRange = 24
	timeFormatLayout = "2006-01-02T15:04:05"
)

type Handler struct {
	service    WalletService
	log        *logrus.Entry
	metrics    *metrics
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

type WalletService interface {
	CreateWallet(ctx context.Context, wallet models.RequestWalletInstance) (models.ResponseWalletInstance, error)
	GetWallet(ctx context.Context, id string) (models.ResponseWalletInstance, error)
	GetWalletsList(ctx context.Context, params models.ListingQueryParams) ([]models.ResponseWalletInstance, error)
	GetWalletHistory(ctx context.Context, id string, params models.RequestWalletHistory) (
		[]models.ResponseWalletHistory, error)
	UpdateWallet(ctx context.Context, wallet models.RequestWalletInstance) (models.ResponseWalletInstance, error)
	DeleteWallet(ctx context.Context, id string) error
	DepositFunds(ctx context.Context, id string, depositFunds models.FundsOperations) (
		models.ResponseWalletInstance, error)
	WithdrawFunds(ctx context.Context, id string, withdrawFunds models.FundsOperations) (
		models.ResponseWalletInstance, error)
	TransferFunds(ctx context.Context, idSrc, idDst string, transferFunds models.FundsOperations) (
		models.ResponseWalletInstance, error)
}

func NewHandler(service WalletService, log *logrus.Logger, privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
) *Handler {
	return &Handler{
		service:    service,
		log:        log.WithField("module", "handler"),
		metrics:    newMetrics(),
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var wallet models.RequestWalletInstance

	err := json.NewDecoder(r.Body).Decode(&wallet)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	wallet.WalletID = uuid.New()

	createdWallet, err := h.service.CreateWallet(r.Context(), wallet)
	if errors.Is(err, postgres.ErrRequestNotIdempotent) || errors.Is(err, postgres.ErrEmailNotUnique) {
		w.WriteHeader(http.StatusConflict)

		return
	}

	if errors.Is(err, walletservice.ErrCurrencyNotValid) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(createdWallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	wallet, err := h.service.GetWallet(r.Context(), id)
	if errors.Is(err, postgres.ErrInvalidWalletID) {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	if errors.Is(err, postgres.ErrWalletNotFound) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(wallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) getList(w http.ResponseWriter, r *http.Request) {
	params := models.ListingQueryParams{}
	params.TextFilter = r.URL.Query().Get("textFilter")

	params.ItemsPerPage, _ = strconv.Atoi(r.URL.Query().Get("itemsPerPage"))
	if params.ItemsPerPage == 0 {
		params.ItemsPerPage = defaultLimit
	}

	params.Offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	params.Sorting = r.URL.Query().Get("sorting")
	params.Descending, _ = strconv.ParseBool(r.URL.Query().Get("descending"))

	walletsList, err := h.service.GetWalletsList(r.Context(), params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(walletsList)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) getHistory(w http.ResponseWriter, r *http.Request) {
	params := models.RequestWalletHistory{}
	params.TextFilter = r.URL.Query().Get("textFilter")

	params.ItemsPerPage, _ = strconv.Atoi(r.URL.Query().Get("itemsPerPage"))
	if params.ItemsPerPage == 0 {
		params.ItemsPerPage = defaultLimit
	}

	params.PeriodStart = time.Now().Add(-defaultTimeRange * time.Hour)
	if r.URL.Query().Get("periodStart") != "" {
		params.PeriodStart, _ = time.Parse(timeFormatLayout, r.URL.Query().Get("periodStart"))
	}

	params.PeriodEnd = time.Now()
	if r.URL.Query().Get("periodEnd") != "" {
		params.PeriodEnd, _ = time.Parse(timeFormatLayout, r.URL.Query().Get("periodEnd"))
	}

	params.Offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	params.Sorting = r.URL.Query().Get("sorting")
	params.Descending, _ = strconv.ParseBool(r.URL.Query().Get("descending"))

	sessionInfo, ok := h.getSessionInfo(r)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	id := sessionInfo.UUID

	walletHistory, err := h.service.GetWalletHistory(r.Context(), id, params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(walletHistory)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var wallet models.RequestWalletInstance

	err := json.NewDecoder(r.Body).Decode(&wallet)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	id := chi.URLParam(r, "id")

	wallet.WalletID = uuid.MustParse(id)

	updatedWallet, err := h.service.UpdateWallet(r.Context(), wallet)
	if errors.Is(err, walletservice.ErrCurrencyNotValid) || errors.Is(err, postgres.ErrWalletNotFound) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if errors.Is(err, postgres.ErrEmailNotUnique) {
		w.WriteHeader(http.StatusConflict)

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(updatedWallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.service.DeleteWallet(r.Context(), id)
	if errors.Is(err, postgres.ErrNoWalletToDelete) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deposit(w http.ResponseWriter, r *http.Request) {
	var depositFunds models.FundsOperations

	err := json.NewDecoder(r.Body).Decode(&depositFunds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	if depositFunds.Amount <= 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	id := chi.URLParam(r, "id")

	updatedWallet, err := h.service.DepositFunds(r.Context(), id, depositFunds)
	if errors.Is(err, postgres.ErrRequestNotIdempotent) {
		w.WriteHeader(http.StatusConflict)

		return
	}

	if errors.Is(err, walletservice.ErrCurrencyNotValid) || errors.Is(err, postgres.ErrWalletNotFound) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(updatedWallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) withdraw(w http.ResponseWriter, r *http.Request) {
	var withdrawFunds models.FundsOperations

	err := json.NewDecoder(r.Body).Decode(&withdrawFunds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	if withdrawFunds.Amount <= 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	id := chi.URLParam(r, "id")

	updatedWallet, err := h.service.WithdrawFunds(r.Context(), id, withdrawFunds)
	if errors.Is(err, walletservice.ErrOverdraft) {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	if errors.Is(err, postgres.ErrRequestNotIdempotent) {
		w.WriteHeader(http.StatusConflict)

		return
	}

	if errors.Is(err, walletservice.ErrCurrencyNotValid) || errors.Is(err, postgres.ErrWalletNotFound) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(updatedWallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) transfer(w http.ResponseWriter, r *http.Request) {
	var transferFunds models.FundsOperations

	err := json.NewDecoder(r.Body).Decode(&transferFunds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	if transferFunds.Amount <= 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	idSrc := chi.URLParam(r, "idSrc")
	idDst := chi.URLParam(r, "idDst")

	dstWallet, err := h.service.TransferFunds(r.Context(), idSrc, idDst, transferFunds)
	if errors.Is(err, walletservice.ErrOverdraft) {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	if errors.Is(err, postgres.ErrRequestNotIdempotent) {
		w.WriteHeader(http.StatusConflict)

		return
	}

	if errors.Is(err, walletservice.ErrCurrencyNotValid) || errors.Is(err, postgres.ErrWalletNotFound) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(dstWallet)
	if err != nil {
		h.log.Warningf("json.NewEncoder.Encode: %s", err)
	}
}

func (h *Handler) getSessionInfo(r *http.Request) (models.SessionInfo, bool) {
	sessionInfo, ok := r.Context().Value(models.SessionInfo{}).(models.SessionInfo)
	if !ok {
		h.log.Warning("err invalid context value")

		return models.SessionInfo{}, false
	}

	return sessionInfo, true
}
