package tests

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/AlexZav1327/service/internal/httpserver"
	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/service"
	"github.com/AlexZav1327/service/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	port                 = ":5005"
	testURL              = "http://localhost" + port
	createWalletEndpoint = "/api/v1/create"
	walletEndpoint       = "/api/v1/wallets"
)

func mainThread(t *testing.T) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	logger := logrus.StandardLogger()
	dsn := os.Getenv("DSN")

	pg, err := postgres.ConnectDB(ctx, logger, dsn)
	require.NoError(t, err)

	err = pg.Migrate(migrate.Up)
	require.NoError(t, err)

	webServer := httpserver.NewServer("", 5005, service.NewWallet(pg, logger), logger)

	err = webServer.Run(ctx)
	require.NoError(t, err)

	err = webServer.Server.Shutdown(ctx)
	require.NoError(t, err)
}

func TestCRUD(t *testing.T) {
	go func() {
		mainThread(t)
	}()

	time.Sleep(250 * time.Millisecond)

	t.Run("create wallet normal cases", func(t *testing.T) {
		creationStatus, createdWallet := sendRequest(testURL+createWalletEndpoint, `{"owner":"John","balance":100.0}`, http.MethodPost, t)

		require.Equal(t, "John", *createdWallet[0].Owner, "they should be equal")
		require.Equal(t, float32(100.0), *createdWallet[0].Balance, "they should be equal")
		require.Equal(t, http.StatusCreated, creationStatus, "they should be equal")
	})

	t.Run("create wallet invalid request data", func(t *testing.T) {
		creationStatus, _ := sendRequest(testURL+createWalletEndpoint, `"owner":Alex,"balance":"200.0"`, http.MethodPost, t)

		require.Equal(t, http.StatusBadRequest, creationStatus, "they should be equal")

		creationStatus, _ = sendRequest(testURL+createWalletEndpoint, `"owner":"Alex","balance":"200.0"`, http.MethodPost, t)

		require.Equal(t, http.StatusBadRequest, creationStatus, "they should be equal")

	})

	t.Run("update wallet normal case", func(t *testing.T) {
		createdWallet := createValidWallet(t)
		walletIdEndpoint := "/" + createdWallet.WalletID.String()

		updateStatus, updatedWallet := sendRequest(testURL+walletEndpoint+walletIdEndpoint, `{"balance":200.5}`, http.MethodPatch, t)

		require.Equal(t, float32(200.5), *updatedWallet[0].Balance, "they should be equal")
		require.Equal(t, http.StatusOK, updateStatus, "they should be equal")
	})

	t.Run("update wallet invalid request data", func(t *testing.T) {
		createdWallet := createValidWallet(t)
		walletIdEndpoint := "/" + createdWallet.WalletID.String()

		updateStatus, _ := sendRequest(testURL+walletEndpoint+walletIdEndpoint, `{"balance": "201"}`, http.MethodPatch, t)

		require.Equal(t, http.StatusBadRequest, updateStatus)
	})

	t.Run("update wallet invalid wallet ID", func(t *testing.T) {
		updateStatus, _ := sendRequest(testURL+walletEndpoint+"/01234567-0123-0123-0123-0123456789ab", `{"balance": 201}`, http.MethodPatch, t)

		require.Equal(t, http.StatusNotFound, updateStatus)
	})

	t.Run("get wallet normal cases", func(t *testing.T) {
		createdWallet := createValidWallet(t)
		walletIdEndpoint := "/" + createdWallet.WalletID.String()

		gettingStatus, receivedWallet := sendRequest(testURL+walletEndpoint+walletIdEndpoint, "", http.MethodGet, t)

		require.Equal(t, *receivedWallet[0].Owner, *createdWallet.Owner, "they should be equal")
		require.Equal(t, *receivedWallet[0].Balance, *createdWallet.Balance, "they should be equal")
		require.Equal(t, http.StatusOK, gettingStatus, "they should be equal")
	})

	t.Run("get wallet invalid wallet ID", func(t *testing.T) {
		gettingStatus, _ := sendRequest(testURL+walletEndpoint+"/01234567-0123-0123-0123-0123456789ab", "", http.MethodGet, t)

		require.Equal(t, http.StatusNotFound, gettingStatus)
	})

	t.Run("delete wallet normal case", func(t *testing.T) {
		createdWallet := createValidWallet(t)
		walletIdEndpoint := "/" + createdWallet.WalletID.String()

		deletionStatus, _ := sendRequest(testURL+walletEndpoint+walletIdEndpoint, "", http.MethodDelete, t)

		require.Equal(t, http.StatusNoContent, deletionStatus, "they should be equal")
	})

	t.Run("delete wallet invalid wallet ID", func(t *testing.T) {
		deletionStatus, _ := sendRequest(testURL+walletEndpoint+"/01234567-0123-0123-0123-0123456789ab", "", http.MethodDelete, t)

		require.Equal(t, http.StatusNotFound, deletionStatus, "they should be equal")
	})

	groomDB(t)
}

func sendRequest(url string, data string, method string, t *testing.T) (int, []models.WalletData) {
	request, err := http.NewRequest(method, url, strings.NewReader(data))
	require.NoError(t, err)

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	respStatus := response.StatusCode

	if respStatus == http.StatusNoContent || respStatus == http.StatusNotFound {
		return respStatus, nil
	}

	body, err := io.ReadAll(response.Body)
	if err != nil && !errors.Is(err, io.EOF) {
		require.NoError(t, err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		require.NoError(t, err)
	}(request.Body)

	var wallet []models.WalletData

	err = json.Unmarshal(body, &wallet)

	if respStatus == http.StatusBadRequest {
		return respStatus, nil
	}

	require.NoError(t, err)

	return respStatus, wallet
}

func createValidWallet(t *testing.T) models.WalletData {
	_, createdWallet := sendRequest(testURL+createWalletEndpoint, `{"owner":"James","balance":300.0}`, http.MethodPost, t)

	return createdWallet[0]
}

func groomDB(t *testing.T) {
	_, walletsList := sendRequest(testURL+walletEndpoint, "", http.MethodGet, t)

	for i := range walletsList {
		_, _ = sendRequest(testURL+walletEndpoint+"/"+walletsList[i].WalletID.String(), "", http.MethodDelete, t)
	}
}
