package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/AlexZav1327/service/internal/httpserver"
	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/service"
	"github.com/AlexZav1327/service/models"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

const (
	port                 = 5005
	createWalletEndpoint = "/api/v1/create"
	walletEndpoint       = "/api/v1/wallets"
	dsn                  = "user=user password=1234 host=localhost port=5432 dbname=postgres sslmode=disable"
)

var testURL = "http://localhost" + ":" + strconv.Itoa(port)

type IntegrationTestSuite struct {
	suite.Suite
	pg        *postgres.Postgres
	webServer *httpserver.Server
	service   *service.Wallet
	models.WalletInstance
	models.ChangeWalletData
	models.WrongWalletData
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	logger := logrus.StandardLogger()

	var err error

	s.pg, err = postgres.ConnectDB(ctx, logger, dsn)
	s.Require().NoError(err)

	err = s.pg.Migrate(migrate.Up)
	s.Require().NoError(err)

	s.service = service.NewWallet(s.pg, logger)

	s.webServer = httpserver.NewServer("", port, s.service, logger)

	go func() {
		_ = s.webServer.Run(ctx)
	}()

	time.Sleep(250 * time.Millisecond)
}

func (s *IntegrationTestSuite) SetupTest() {
	s.WalletInstance = models.WalletInstance{
		WalletID: uuid.MustParse("01234567-0123-0123-0123-0123456789a5"),
		Owner:    "Alex",
		Balance:  6666,
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	s.ChangeWalletData = models.ChangeWalletData{
		WalletID: uuid.MustParse("01234567-0123-0123-0123-0123456789a5"),
		Owner:    "Liza",
		Balance:  7569,
	}

	s.WrongWalletData = models.WrongWalletData{
		WalletID: uuid.MustParse("01234567-0123-0123-0123-0123456789a5"),
		Balance:  "1050",
	}
}

func (s *IntegrationTestSuite) TearDownSuite() {
	ctx := context.Background()

	var err error

	err = s.pg.ResetTable(ctx)
	s.Require().NoError(err)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) sendRequest(ctx context.Context, method, endpoint string, body interface{}, dest interface{}) *http.Response {
	s.T().Helper()

	reqBody, err := json.Marshal(body)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(reqBody))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)

	defer func() {
		err = resp.Body.Close()
		s.Require().NoError(err)
	}()

	if dest != nil {
		err = json.NewDecoder(resp.Body).Decode(&dest)
		s.Require().NoError(err)
	}

	return resp
}

func (s *IntegrationTestSuite) TestWalletCRUD() {
	s.Run("create wallet normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		resp := s.sendRequest(ctx, http.MethodPost, testURL+createWalletEndpoint, s.WalletInstance, &respData)

		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(s.WalletInstance.Owner, respData.Owner)
		s.Require().Equal(s.WalletInstance.Balance, respData.Balance)
	})

	s.Run("create wallet invalid wallet balance", func() {
		ctx := context.Background()
		resp := s.sendRequest(ctx, http.MethodPost, testURL+createWalletEndpoint, s.WrongWalletData, nil)

		s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	})

	s.Run("get wallet normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpoint := "/" + s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodGet, testURL+walletEndpoint+walletIdEndpoint, s.WalletInstance, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.WalletInstance.Owner, respData.Owner)
		s.Require().Equal(s.WalletInstance.Balance, respData.Balance)
	})

	s.Run("get a list of wallets normal case", func() {
		ctx := context.Background()

		var respData []models.WalletInstance

		resp := s.sendRequest(ctx, http.MethodGet, testURL+walletEndpoint, s.WalletInstance, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
	})

	s.Run("get wallet invalid wallet ID", func() {
		ctx := context.Background()
		walletIdEndpoint := "/" + uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodGet, testURL+walletEndpoint+walletIdEndpoint, s.WalletInstance, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("update wallet normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpoint := "/" + s.ChangeWalletData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPatch, testURL+walletEndpoint+walletIdEndpoint, s.ChangeWalletData, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.ChangeWalletData.Owner, respData.Owner)
		s.Require().Equal(s.ChangeWalletData.Balance, respData.Balance)
	})

	s.Run("update wallet invalid wallet ID", func() {
		ctx := context.Background()

		walletIdEndpoint := "/" + uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodPatch, testURL+walletEndpoint+walletIdEndpoint, s.ChangeWalletData, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("update wallet invalid wallet balance", func() {
		ctx := context.Background()

		walletIdEndpoint := "/" + s.WrongWalletData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPatch, testURL+walletEndpoint+walletIdEndpoint, s.WrongWalletData, nil)

		s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
	})

	s.Run("delete wallet normal case", func() {
		ctx := context.Background()
		walletIdEndpoint := "/" + s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodDelete, testURL+walletEndpoint+walletIdEndpoint, s.WalletInstance, nil)

		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	})

	s.Run("delete wallet invalid wallet ID", func() {
		ctx := context.Background()
		walletIdEndpoint := "/" + uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodDelete, testURL+walletEndpoint+walletIdEndpoint, s.WalletInstance, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})
}
