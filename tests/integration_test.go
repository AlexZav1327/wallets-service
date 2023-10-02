package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/AlexZav1327/service/internal/postgres"
	walletserver "github.com/AlexZav1327/service/internal/wallet-server"
	walletservice "github.com/AlexZav1327/service/internal/wallet-service"
	"github.com/AlexZav1327/service/models"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

const (
	port                 = 5005
	createWalletEndpoint = "/api/v1/wallet/create"
	walletEndpoint       = "/api/v1/wallet/"
	walletsEndpoint      = "/api/v1/wallets"
	dsn                  = "user=user password=1234 host=localhost port=5432 dbname=postgres sslmode=disable"
	deposit              = "/deposit"
	withdraw             = "/withdraw"
	transfer             = "/transfer/"
)

var url = fmt.Sprintf("http://localhost:%d", port)

type IntegrationTestSuite struct {
	suite.Suite
	pg            *postgres.Postgres
	server        *walletserver.Server
	walletService *walletservice.Service
	models.RequestWalletInstance
	models.FundsOperations
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	logger := logrus.StandardLogger()

	var err error

	s.pg, err = postgres.ConnectDB(ctx, logger, dsn)
	s.Require().NoError(err)

	err = s.pg.Migrate(migrate.Up)
	s.Require().NoError(err)

	s.walletService = walletservice.New(s.pg, logger)

	s.server = walletserver.New("", port, s.walletService, logger)

	go func() {
		_ = s.server.Run(ctx)
	}()

	time.Sleep(250 * time.Millisecond)
}

func (s *IntegrationTestSuite) SetupTest() {
	s.RequestWalletInstance = models.RequestWalletInstance{
		TransactionKey: uuid.MustParse("76543210-3210-3210-3210-012345678aa1"),
		WalletID:       uuid.MustParse("01234567-0123-0123-0123-0123456789aa"),
		Owner:          "Kate",
		Currency:       "EUR",
		Balance:        500,
	}

	s.FundsOperations = models.FundsOperations{
		TransactionKey: uuid.MustParse("76543210-3210-3210-3210-012345678aa2"),
		Currency:       "EUR",
		Amount:         200,
	}
}

func (s *IntegrationTestSuite) TearDownTest() {
	ctx := context.Background()

	err := s.pg.TruncateTable(ctx, "wallet")
	s.Require().NoError(err)

	err = s.pg.TruncateTable(ctx, "idempotency")
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

func (s *IntegrationTestSuite) TestCRUD() {
	s.Run("create wallet normal case", func() {
		ctx := context.Background()

		var respData models.ResponseWalletInstance

		resp := s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, s.RequestWalletInstance, &respData)

		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(s.RequestWalletInstance.WalletID, respData.WalletID)
		s.Require().Equal(s.RequestWalletInstance.Owner, respData.Owner)
		s.Require().Equal(s.RequestWalletInstance.Currency, respData.Currency)
		s.Require().Equal(float32(0), respData.Balance)
	})

	s.Run("create wallet invalid currency", func() {
		ctx := context.Background()
		wallet := s.RequestWalletInstance

		wallet.TransactionKey = uuid.New()
		wallet.Currency = "XYZ"

		resp := s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, wallet, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("create wallet not unique transaction key", func() {
		ctx := context.Background()
		resp := s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, s.RequestWalletInstance, nil)

		s.Require().Equal(http.StatusConflict, resp.StatusCode)
	})

	s.Run("get wallet normal case", func() {
		ctx := context.Background()

		var respData models.ResponseWalletInstance

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint, nil, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.RequestWalletInstance.Owner, respData.Owner)
		s.Require().Equal(s.RequestWalletInstance.Currency, respData.Currency)
		s.Require().Equal(float32(0), respData.Balance)
	})

	s.Run("get wallet invalid wallet ID", func() {
		ctx := context.Background()
		walletIdEndpoint := uuid.New().String()

		resp := s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("get a list of wallets normal case", func() {
		ctx := context.Background()

		resp := s.sendRequest(ctx, http.MethodGet, url+walletsEndpoint, nil, nil)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
	})

	s.Run("update wallet currency normal case", func() {
		ctx := context.Background()

		_, err := s.pg.ManageBalance(ctx, s.RequestWalletInstance.WalletID.String(), s.RequestWalletInstance.Balance)
		s.Require().NoError(err)

		wallet := s.RequestWalletInstance

		wallet.Currency = "USD"

		var respData models.ResponseWalletInstance

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPatch, url+walletEndpoint+walletIdEndpoint, wallet, &respData)

		convertedCurrentCurrencyFunds, _ := s.walletService.ConvertCurrency(ctx, s.RequestWalletInstance.Currency, wallet.Currency, s.RequestWalletInstance.Balance)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(convertedCurrentCurrencyFunds, respData.Balance)
	})

	s.Run("update wallet invalid currency", func() {
		ctx := context.Background()

		wallet := s.RequestWalletInstance

		wallet.Currency = "XYZ"

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPatch, url+walletEndpoint+walletIdEndpoint, wallet, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("delete wallet normal case", func() {
		ctx := context.Background()
		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodDelete, url+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	})

	s.Run("delete wallet invalid wallet ID", func() {
		ctx := context.Background()
		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()

		resp := s.sendRequest(ctx, http.MethodDelete, url+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestDeposit() {
	s.Run("deposit funds current currency normal case", func() {
		ctx := context.Background()

		_, err := s.pg.CreateWallet(ctx, s.RequestWalletInstance)
		s.Require().NoError(err)

		_, err = s.pg.ManageBalance(ctx, s.RequestWalletInstance.WalletID.String(), s.RequestWalletInstance.Balance)
		s.Require().NoError(err)

		var respData models.ResponseWalletInstance

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, s.FundsOperations, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.RequestWalletInstance.Balance+s.FundsOperations.Amount, respData.Balance)
	})

	s.Run("deposit funds different currency normal case", func() {
		ctx := context.Background()
		transaction := s.FundsOperations

		transaction.TransactionKey = uuid.New()
		transaction.Currency = "RUB"
		transaction.Amount = 10000

		var respData models.ResponseWalletInstance

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, transaction, &respData)

		convertedDifferentCurrencyFunds, _ := s.walletService.ConvertCurrency(ctx, transaction.Currency, s.RequestWalletInstance.Currency, transaction.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.RequestWalletInstance.Balance+s.FundsOperations.Amount+convertedDifferentCurrencyFunds, respData.Balance)
	})

	s.Run("deposit funds invalid currency", func() {
		ctx := context.Background()
		transaction := s.FundsOperations

		transaction.TransactionKey = uuid.New()
		transaction.Currency = "XYZ"

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, transaction, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("deposit funds negative value", func() {
		ctx := context.Background()
		transaction := s.FundsOperations

		transaction.TransactionKey = uuid.New()
		transaction.Amount = -100

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, transaction, nil)

		s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("deposit funds not unique transaction key", func() {
		ctx := context.Background()

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, s.FundsOperations, nil)

		s.Require().Equal(http.StatusConflict, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestWithdraw() {
	s.Run("withdraw funds current currency normal case", func() {
		ctx := context.Background()

		_, err := s.pg.CreateWallet(ctx, s.RequestWalletInstance)
		s.Require().NoError(err)

		_, err = s.pg.ManageBalance(ctx, s.RequestWalletInstance.WalletID.String(), s.RequestWalletInstance.Balance)
		s.Require().NoError(err)

		var respData models.ResponseWalletInstance

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, s.FundsOperations, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.RequestWalletInstance.Balance-s.FundsOperations.Amount, respData.Balance)
	})

	s.Run("withdraw funds different currency normal case", func() {
		ctx := context.Background()
		transaction := s.FundsOperations

		transaction.TransactionKey = uuid.New()
		transaction.Currency = "RUB"
		transaction.Amount = 1000

		var respData models.ResponseWalletInstance

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, transaction, &respData)

		convertedDifferentCurrencyFunds, _ := s.walletService.ConvertCurrency(ctx, transaction.Currency, s.RequestWalletInstance.Currency, transaction.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.RequestWalletInstance.Balance-s.FundsOperations.Amount-convertedDifferentCurrencyFunds, respData.Balance)
	})

	s.Run("withdraw funds overdraft", func() {
		ctx := context.Background()
		transaction := s.FundsOperations

		transaction.TransactionKey = uuid.New()
		transaction.Currency = "RUB"
		transaction.Amount = 100000

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, transaction, nil)

		s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("withdraw funds invalid currency", func() {
		ctx := context.Background()
		transaction := s.FundsOperations

		transaction.TransactionKey = uuid.New()
		transaction.Currency = "XYZ"

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, transaction, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("withdraw funds not unique transaction key", func() {
		ctx := context.Background()

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, s.FundsOperations, nil)

		s.Require().Equal(http.StatusConflict, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestTransfer() {
	s.Run("transfer funds normal case", func() {
		ctx := context.Background()

		_, err := s.pg.CreateWallet(ctx, s.RequestWalletInstance)
		s.Require().NoError(err)

		_, err = s.pg.ManageBalance(ctx, s.RequestWalletInstance.WalletID.String(), s.RequestWalletInstance.Balance)
		s.Require().NoError(err)

		walletDst := s.RequestWalletInstance

		walletDst.TransactionKey = uuid.New()
		walletDst.WalletID = uuid.New()
		walletDst.Currency = "USD"
		walletDst.Balance = 350

		_, err = s.pg.CreateWallet(ctx, walletDst)
		s.Require().NoError(err)

		_, err = s.pg.ManageBalance(ctx, walletDst.WalletID.String(), walletDst.Balance)
		s.Require().NoError(err)

		var respData models.ResponseWalletInstance

		walletIdEndpointSrc := s.RequestWalletInstance.WalletID.String()
		walletIdEndpointDst := walletDst.WalletID.String()

		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpointSrc+transfer+walletIdEndpointDst, s.FundsOperations, &respData)

		convertedTransferredFunds, _ := s.walletService.ConvertCurrency(ctx, s.FundsOperations.Currency, walletDst.Currency, s.FundsOperations.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(walletDst.Balance+convertedTransferredFunds, respData.Balance)
	})

	s.Run("transfer funds invalid destination wallet ID", func() {
		ctx := context.Background()

		s.RequestWalletInstance.TransactionKey = uuid.New()
		s.RequestWalletInstance.WalletID = uuid.New()

		_, err := s.pg.CreateWallet(ctx, s.RequestWalletInstance)
		s.Require().NoError(err)

		_, err = s.pg.ManageBalance(ctx, s.RequestWalletInstance.WalletID.String(), s.RequestWalletInstance.Balance)
		s.Require().NoError(err)

		walletDst := s.RequestWalletInstance

		walletDst.TransactionKey = uuid.New()
		walletDst.WalletID = uuid.New()
		walletDst.Currency = "USD"
		walletDst.Balance = 350

		_, err = s.pg.CreateWallet(ctx, walletDst)
		s.Require().NoError(err)

		_, err = s.pg.ManageBalance(ctx, walletDst.WalletID.String(), walletDst.Balance)
		s.Require().NoError(err)

		transaction := s.FundsOperations

		transaction.TransactionKey = uuid.New()

		walletIdEndpoint := s.RequestWalletInstance.WalletID.String()
		walletIdEndpointDst := uuid.New().String()

		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+transfer+walletIdEndpointDst, transaction, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
		s.Require().Equal(s.RequestWalletInstance.Balance, float32(500))
		s.Require().Equal(walletDst.Balance, float32(350))
	})

	s.Run("transfer funds not unique transaction key", func() {
		ctx := context.Background()

		walletDst := s.RequestWalletInstance

		walletDst.TransactionKey = uuid.New()
		walletDst.WalletID = uuid.New()

		walletIdEndpointSrc := s.RequestWalletInstance.WalletID.String()
		walletIdEndpointDst := walletDst.WalletID.String()

		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpointSrc+transfer+walletIdEndpointDst, s.FundsOperations, nil)

		s.Require().Equal(http.StatusConflict, resp.StatusCode)
	})
}
