package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/AlexZav1327/service/internal/postgres"
	walletserver "github.com/AlexZav1327/service/internal/wallet-server"
	walletservice "github.com/AlexZav1327/service/internal/wallet-service"
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
	history              = "/history"
)

var url = fmt.Sprintf("http://localhost:%d", port)

type IntegrationTestSuite struct {
	suite.Suite
	pg            *postgres.Postgres
	server        *walletserver.Server
	walletService *walletservice.Service
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

func (s *IntegrationTestSuite) SetupTest() {}

func (s *IntegrationTestSuite) TearDownTest() {
	ctx := context.Background()

	err := s.pg.TruncateTable(ctx, "wallet")
	s.Require().NoError(err)

	err = s.pg.TruncateTable(ctx, "idempotency")
	s.Require().NoError(err)

	err = s.pg.TruncateTable(ctx, "history")
	s.Require().NoError(err)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) sendRequest(ctx context.Context, method, endpoint string, body interface{},
	dest interface{},
) *http.Response {
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

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "EUR"
		req.Balance = 350

		var respData models.ResponseWalletInstance

		resp := s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(req.Owner, respData.Owner)
		s.Require().Equal(req.Currency, respData.Currency)
		s.Require().Equal(float32(0), respData.Balance)
	})

	s.Run("create wallet not valid currency", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "XYZ"

		resp := s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("create wallet non-idempotent request", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "USD"

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, nil)
		resp := s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, nil)

		s.Require().Equal(http.StatusConflict, resp.StatusCode)
	})

	s.Run("get wallet normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint, nil, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(req.Owner, respData.Owner)
		s.Require().Equal(req.Currency, respData.Currency)
	})

	s.Run("get wallet not valid wallet ID", func() {
		ctx := context.Background()

		walletIdEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("update wallet normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Liza"
		req.Currency = "EUR"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "EUR"
		reqDeposit.Amount = 100

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, &respData)

		reqUpdate := req
		reqUpdate.Owner = "Alex"
		reqUpdate.Currency = "USD"

		resp := s.sendRequest(ctx, http.MethodPatch, url+walletEndpoint+walletIdEndpoint, reqUpdate, &respData)

		convertedFunds, _ := s.walletService.ConvertCurrency(ctx, req.Currency, reqUpdate.Currency, reqDeposit.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(reqUpdate.Owner, respData.Owner)
		s.Require().Equal(reqUpdate.Currency, respData.Currency)
		s.Require().Equal(convertedFunds, respData.Balance)
	})

	s.Run("update wallet not valid currency", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Liza"
		req.Currency = "EUR"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqUpdate := req
		reqUpdate.Currency = "XYZ"

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPatch, url+walletEndpoint+walletIdEndpoint, reqUpdate, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("update wallet not valid wallet ID", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.Currency = "RUB"

		walletIdEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodPatch, url+walletEndpoint+walletIdEndpoint, req, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("delete wallet normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodDelete, url+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	})

	s.Run("delete wallet not valid wallet ID", func() {
		ctx := context.Background()

		walletIdEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodDelete, url+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestDeposit() {
	s.Run("deposit funds current currency normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(reqDeposit.Amount, respData.Balance)
	})

	s.Run("deposit funds different currency normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, &respData)

		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "EUR"
		resp = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, &respData)

		convertedFunds, _ := s.walletService.ConvertCurrency(ctx, reqDeposit.Currency, req.Currency, reqDeposit.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(reqDeposit.Amount+convertedFunds, respData.Balance)
	})

	s.Run("deposit funds non-idempotent request", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "USD"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = req.TransactionKey
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		s.Require().Equal(http.StatusConflict, resp.StatusCode)
	})

	s.Run("deposit funds not valid currency", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "USD"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "XYZ"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("deposit funds non-positive amount value", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "USD"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "EUR"
		reqDeposit.Amount = 0

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestWithdraw() {
	s.Run("withdraw funds current currency normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		reqWithdraw := models.FundsOperations{}
		reqWithdraw.TransactionKey = uuid.New()
		reqWithdraw.Currency = "RUB"
		reqWithdraw.Amount = 200

		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, reqWithdraw, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(reqDeposit.Amount-reqWithdraw.Amount, respData.Balance)
	})

	s.Run("withdraw funds different currency normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		reqWithdraw := models.FundsOperations{}
		reqWithdraw.TransactionKey = uuid.New()
		reqWithdraw.Currency = "USD"
		reqWithdraw.Amount = 1

		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, reqWithdraw, &respData)

		convertedFunds, _ := s.walletService.ConvertCurrency(ctx, reqWithdraw.Currency, reqDeposit.Currency, reqWithdraw.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(reqDeposit.Amount-convertedFunds, respData.Balance)
	})

	s.Run("withdraw funds non-idempotent request", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		reqWithdraw := models.FundsOperations{}
		reqWithdraw.TransactionKey = reqDeposit.TransactionKey
		reqWithdraw.Currency = "RUB"
		reqWithdraw.Amount = 1200

		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, reqWithdraw, nil)

		s.Require().Equal(http.StatusConflict, resp.StatusCode)
	})

	s.Run("withdraw funds overdraft", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		reqWithdraw := models.FundsOperations{}
		reqWithdraw.TransactionKey = uuid.New()
		reqWithdraw.Currency = "RUB"
		reqWithdraw.Amount = 1200

		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, reqWithdraw, nil)

		s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("withdraw funds not valid currency", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, &respData)

		reqWithdraw := models.FundsOperations{}
		reqWithdraw.TransactionKey = uuid.New()
		reqWithdraw.Currency = "XYZ"
		reqWithdraw.Amount = 200

		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, reqWithdraw, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestTransfer() {
	s.Run("transfer funds normal case", func() {
		ctx := context.Background()

		reqSrcWallet := models.RequestWalletInstance{}
		reqSrcWallet.TransactionKey = uuid.New()
		reqSrcWallet.Owner = "Alex"
		reqSrcWallet.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, reqSrcWallet, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 10000

		srcWalletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+srcWalletIdEndpoint+deposit, reqDeposit, nil)

		reqDstWallet := models.RequestWalletInstance{}
		reqDstWallet.TransactionKey = uuid.New()
		reqDstWallet.Owner = "Kate"
		reqDstWallet.Currency = "USD"

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, reqDstWallet, &respData)

		reqTransfer := models.FundsOperations{}
		reqTransfer.TransactionKey = uuid.New()
		reqTransfer.Currency = "RUB"
		reqTransfer.Amount = 9999

		dstWalletEndpoint := respData.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+srcWalletIdEndpoint+transfer+dstWalletEndpoint,
			reqTransfer, &respData)

		convertedFunds, _ := s.walletService.ConvertCurrency(ctx, reqSrcWallet.Currency, reqDstWallet.Currency,
			reqTransfer.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(convertedFunds, respData.Balance)

		_ = s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+srcWalletIdEndpoint, nil, &respData)

		s.Require().Equal(reqDeposit.Amount-reqTransfer.Amount, respData.Balance)
	})

	s.Run("transfer funds non-idempotent request", func() {
		ctx := context.Background()

		reqSrcWallet := models.RequestWalletInstance{}
		reqSrcWallet.TransactionKey = uuid.New()
		reqSrcWallet.Owner = "Alex"
		reqSrcWallet.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, reqSrcWallet, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 10000

		srcWalletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+srcWalletIdEndpoint+deposit, reqDeposit, nil)

		reqTransfer := models.FundsOperations{}
		reqTransfer.TransactionKey = reqDeposit.TransactionKey
		reqTransfer.Currency = "RUB"
		reqTransfer.Amount = 9999

		dstWalletEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+srcWalletIdEndpoint+transfer+dstWalletEndpoint,
			reqTransfer, nil)

		_ = s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+srcWalletIdEndpoint, nil, &respData)

		s.Require().Equal(http.StatusConflict, resp.StatusCode)
		s.Require().Equal(reqDeposit.Amount, respData.Balance)
	})

	s.Run("transfer funds not valid destination wallet ID", func() {
		ctx := context.Background()

		reqSrcWallet := models.RequestWalletInstance{}
		reqSrcWallet.TransactionKey = uuid.New()
		reqSrcWallet.Owner = "Alex"
		reqSrcWallet.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, reqSrcWallet, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "RUB"
		reqDeposit.Amount = 10000

		srcWalletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+srcWalletIdEndpoint+deposit, reqDeposit, nil)

		reqTransfer := models.FundsOperations{}
		reqTransfer.TransactionKey = uuid.New()
		reqTransfer.Currency = "RUB"
		reqTransfer.Amount = 9999

		dstWalletEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+srcWalletIdEndpoint+transfer+dstWalletEndpoint,
			reqTransfer, nil)

		_ = s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+srcWalletIdEndpoint, nil, &respData)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
		s.Require().Equal(reqDeposit.Amount, respData.Balance)
	})
}

func (s *IntegrationTestSuite) TestWalletsList() {
	s.Run("get empty list of wallets normal case", func() {
		ctx := context.Background()

		var respData []models.ResponseWalletInstance

		resp := s.sendRequest(ctx, http.MethodGet, url+walletsEndpoint, nil, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal([]models.ResponseWalletInstance{}, respData)
	})

	s.Run("get list of wallets normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "RUB"

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, nil)

		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "USD"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "USD"
		reqDeposit.Amount = 100

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		req.TransactionKey = uuid.New()
		req.Owner = "Liza"
		req.Currency = "EUR"

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, nil)

		var respDataList []models.ResponseWalletInstance

		resp := s.sendRequest(ctx, http.MethodGet, url+walletsEndpoint, nil, &respDataList)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(3, len(respDataList))

		queryParams := "?textFilter=Alex"
		resp = s.sendRequest(ctx, http.MethodGet, url+walletsEndpoint+queryParams, nil, &respDataList)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(1, len(respDataList))
		s.Require().Equal("Alex", respDataList[0].Owner)

		queryParams = "?sorting=balance&descending=true"
		resp = s.sendRequest(ctx, http.MethodGet, url+walletsEndpoint+queryParams, nil, &respDataList)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(float32(100), respDataList[0].Balance)

		queryParams = "?itemsPerPage=2"
		resp = s.sendRequest(ctx, http.MethodGet, url+walletsEndpoint+queryParams, nil, &respDataList)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(2, len(respDataList))

		queryParams = "?offset=1"
		resp = s.sendRequest(ctx, http.MethodGet, url+walletsEndpoint+queryParams, nil, &respDataList)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(2, len(respDataList))
	})
}

func (s *IntegrationTestSuite) TestHistory() {
	s.Run("get wallet history normal case", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "USD"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "USD"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		reqWithdraw := models.FundsOperations{}
		reqWithdraw.TransactionKey = uuid.New()
		reqWithdraw.Currency = "USD"
		reqWithdraw.Amount = 150

		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+withdraw, reqWithdraw, nil)

		reqUpdate := req
		reqUpdate.Owner = "Noname"
		reqUpdate.Currency = "EUR"

		_ = s.sendRequest(ctx, http.MethodPatch, url+walletEndpoint+walletIdEndpoint, reqUpdate, nil)

		_ = s.sendRequest(ctx, http.MethodDelete, url+walletEndpoint+walletIdEndpoint, nil, nil)

		var respDataHistory []models.ResponseWalletHistory

		queryParams := fmt.Sprintf("?periodStart=%s&periodEnd=%s",
			time.Now().Add(-12*time.Hour).Format("2006-01-02T15:04:05"),
			time.Now().Format("2006-01-02T15:04:05"))

		resp := s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint+history+queryParams, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(5, len(respDataHistory))

		queryParams = "?textFilter=Noname"
		resp = s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint+history+queryParams, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(2, len(respDataHistory))
		s.Require().Equal("Noname", respDataHistory[0].Owner)

		queryParams = "?sorting=balance&descending=true"
		resp = s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint+history+queryParams, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(float32(1000), respDataHistory[0].Balance)

		queryParams = "?itemsPerPage=3"
		resp = s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint+history+queryParams, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(3, len(respDataHistory))

		queryParams = "?offset=2"
		resp = s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint+history+queryParams, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(3, len(respDataHistory))
	})

	s.Run("get wallet history non-active period", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Alex"
		req.Currency = "USD"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		reqDeposit := models.FundsOperations{}
		reqDeposit.TransactionKey = uuid.New()
		reqDeposit.Currency = "USD"
		reqDeposit.Amount = 1000

		walletIdEndpoint := respData.WalletID.String()
		_ = s.sendRequest(ctx, http.MethodPut, url+walletEndpoint+walletIdEndpoint+deposit, reqDeposit, nil)

		var respDataHistory []models.ResponseWalletHistory

		queryParams := fmt.Sprintf("?periodStart=%s&periodEnd=%s",
			time.Now().Format("2006-01-02T15:04:05"),
			time.Now().Add(3*time.Second).Format("2006-01-02T15:04:05"))

		resp := s.sendRequest(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint+history+queryParams, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal([]models.ResponseWalletHistory{}, respDataHistory)
	})
}
