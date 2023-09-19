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

var testURL = fmt.Sprintf("http://localhost:%d", port)

type IntegrationTestSuite struct {
	suite.Suite
	pg            *postgres.Postgres
	server        *walletserver.Server
	walletService *walletservice.Service
	models.WalletInstance
	WalletInstanceDst         models.WalletInstance
	CreateValidWallet         models.WalletInstance
	CreateWrongWallet         models.WalletInstance
	TransactCurrentCurrency   models.ManageFunds
	TransactDifferentCurrency models.ManageFunds
	TransactWrongCurrency     models.ManageFunds
	TransactOverdraft         models.ManageFunds
	TransactNegativeValue     models.ManageFunds
	ValidChangeCurrency       models.ChangeCurrency
	WrongChangeCurrency       models.ChangeCurrency
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
	s.WalletInstance = models.WalletInstance{
		WalletID: uuid.MustParse("01234567-0123-0123-0123-0123456789aa"),
		Owner:    "Kate",
		Currency: "EUR",
		Balance:  500,
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	s.WalletInstanceDst = models.WalletInstance{
		WalletID: uuid.MustParse("01234567-0123-0123-0123-0123456789bb"),
		Owner:    "Alex",
		Currency: "RUB",
		Balance:  2500,
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	ctx := context.Background()

	_, err := s.pg.CreateWallet(ctx, s.WalletInstance)
	s.Require().NoError(err)

	_, err = s.pg.ManageBalance(ctx, s.WalletInstance.WalletID.String(), s.WalletInstance.Balance)
	s.Require().NoError(err)

	_, err = s.pg.CreateWallet(ctx, s.WalletInstanceDst)
	s.Require().NoError(err)

	_, err = s.pg.ManageBalance(ctx, s.WalletInstanceDst.WalletID.String(), s.WalletInstanceDst.Balance)
	s.Require().NoError(err)

	s.CreateValidWallet = models.WalletInstance{
		WalletID: uuid.MustParse("01234567-0123-0123-0123-0123456789ab"),
		Owner:    "Liza",
		Currency: "EUR",
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	s.CreateWrongWallet = models.WalletInstance{
		WalletID: uuid.MustParse("01234567-0123-0123-0123-0123456789ba"),
		Owner:    "Liza",
		Currency: "X",
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	s.TransactCurrentCurrency = models.ManageFunds{
		Currency: "EUR",
		Amount:   200,
	}

	s.TransactDifferentCurrency = models.ManageFunds{
		Currency: "USD",
		Amount:   300,
	}

	s.TransactWrongCurrency = models.ManageFunds{
		Currency: "X",
		Amount:   100,
	}

	s.TransactOverdraft = models.ManageFunds{
		Currency: "USD",
		Amount:   10000,
	}

	s.TransactNegativeValue = models.ManageFunds{
		Currency: "USD",
		Amount:   -10,
	}

	s.ValidChangeCurrency = models.ChangeCurrency{
		Currency: "USD",
	}

	s.WrongChangeCurrency = models.ChangeCurrency{
		Currency: "X",
	}
}

func (s *IntegrationTestSuite) TearDownTest() {
	ctx := context.Background()

	err := s.pg.TruncateTable(ctx, "wallet")
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

		var respData models.WalletInstance

		resp := s.sendRequest(ctx, http.MethodPost, testURL+createWalletEndpoint, s.CreateValidWallet, &respData)

		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(s.CreateValidWallet.Owner, respData.Owner)
		s.Require().Equal(s.CreateValidWallet.Currency, respData.Currency)
	})

	s.Run("create wallet invalid wallet data", func() {
		ctx := context.Background()
		resp := s.sendRequest(ctx, http.MethodPost, testURL+createWalletEndpoint, s.CreateWrongWallet, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("get wallet normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodGet, testURL+walletEndpoint+walletIdEndpoint, nil, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.WalletInstance.Owner, respData.Owner)
		s.Require().Equal(s.WalletInstance.Currency, respData.Currency)
		s.Require().Equal(s.WalletInstance.Balance, respData.Balance)
	})

	s.Run("get wallet invalid wallet ID", func() {
		ctx := context.Background()
		walletIdEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodGet, testURL+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("get a list of wallets normal case", func() {
		ctx := context.Background()

		resp := s.sendRequest(ctx, http.MethodGet, testURL+walletsEndpoint, nil, nil)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
	})

	s.Run("update wallet currency normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPatch, testURL+walletEndpoint+walletIdEndpoint, s.ValidChangeCurrency, &respData)

		convertedCurrentCurrencyFunds, _ := s.walletService.ConvertCurrency(ctx, s.WalletInstance.Currency, s.ValidChangeCurrency.Currency, s.WalletInstance.Balance)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(convertedCurrentCurrencyFunds, respData.Balance)
	})

	s.Run("update wallet invalid currency", func() {
		ctx := context.Background()

		walletIdEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodPatch, testURL+walletEndpoint+walletIdEndpoint, s.WrongChangeCurrency, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("delete wallet normal case", func() {
		ctx := context.Background()
		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodDelete, testURL+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	})

	s.Run("delete wallet invalid wallet ID", func() {
		ctx := context.Background()
		walletIdEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodDelete, testURL+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestDeposit() {
	s.Run("deposit funds current currency normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+deposit, s.TransactCurrentCurrency, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.WalletInstance.Balance+s.TransactCurrentCurrency.Amount, respData.Balance)
	})

	s.Run("deposit funds different currency normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+deposit, s.TransactDifferentCurrency, &respData)

		convertedDifferentCurrencyFunds, _ := s.walletService.ConvertCurrency(ctx, s.TransactDifferentCurrency.Currency, s.WalletInstance.Currency, s.TransactDifferentCurrency.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.WalletInstance.Balance+s.TransactCurrentCurrency.Amount+convertedDifferentCurrencyFunds, respData.Balance)
	})

	s.Run("deposit funds invalid currency", func() {
		ctx := context.Background()

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+deposit, s.TransactWrongCurrency, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("deposit funds negative value", func() {
		ctx := context.Background()

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+deposit, s.TransactNegativeValue, nil)

		s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestWithdraw() {
	s.Run("withdraw funds current currency normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+withdraw, s.TransactCurrentCurrency, &respData)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.WalletInstance.Balance-s.TransactCurrentCurrency.Amount, respData.Balance)
	})

	s.Run("withdraw funds different currency normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+withdraw, s.TransactDifferentCurrency, &respData)

		convertedDifferentCurrencyFunds, _ := s.walletService.ConvertCurrency(ctx, s.TransactDifferentCurrency.Currency, s.WalletInstance.Currency, s.TransactDifferentCurrency.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.WalletInstance.Balance-s.TransactCurrentCurrency.Amount-convertedDifferentCurrencyFunds, respData.Balance)
	})

	s.Run("withdraw funds overdraft", func() {
		ctx := context.Background()

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+withdraw, s.TransactOverdraft, nil)

		s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
	})

	s.Run("withdraw funds invalid currency", func() {
		ctx := context.Background()

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+withdraw, s.TransactWrongCurrency, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})
}

func (s *IntegrationTestSuite) TestTransfer() {
	s.Run("transfer funds normal case", func() {
		ctx := context.Background()

		var respData models.WalletInstance

		walletIdEndpointSrc := s.WalletInstance.WalletID.String()
		walletIdEndpointDst := s.WalletInstanceDst.WalletID.String()

		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpointSrc+transfer+walletIdEndpointDst, s.TransactCurrentCurrency, &respData)

		convertedTransferredFunds, _ := s.walletService.ConvertCurrency(ctx, s.TransactCurrentCurrency.Currency, s.WalletInstanceDst.Currency, s.TransactCurrentCurrency.Amount)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(s.WalletInstanceDst.Balance+convertedTransferredFunds, respData.Balance)
	})

	s.Run("transfer funds invalid destination wallet ID", func() {
		ctx := context.Background()

		walletIdEndpoint := s.WalletInstance.WalletID.String()
		walletIdEndpointDst := uuid.New().String()

		resp := s.sendRequest(ctx, http.MethodPut, testURL+walletEndpoint+walletIdEndpoint+transfer+walletIdEndpointDst, s.TransactCurrentCurrency, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
		s.Require().Equal(s.WalletInstance.Balance, float32(500))
		s.Require().Equal(s.WalletInstanceDst.Balance, float32(2500))
	})
}
