package tests

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/google/uuid"
)

func (s *IntegrationTestSuite) TestWalletCRUD() {
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

		resp := s.sendRequest(ctx, http.MethodPatch, url+updateWalletEndpoint+walletIdEndpoint, reqUpdate, &respData)

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
		resp := s.sendRequest(ctx, http.MethodPatch, url+updateWalletEndpoint+walletIdEndpoint, reqUpdate, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
	})

	s.Run("update wallet not valid wallet ID", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.Currency = "RUB"

		walletIdEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodPatch, url+updateWalletEndpoint+walletIdEndpoint, req, nil)

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
		resp := s.sendRequest(ctx, http.MethodDelete, url+deleteWalletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNoContent, resp.StatusCode)
	})

	s.Run("delete wallet not valid wallet ID", func() {
		ctx := context.Background()

		walletIdEndpoint := uuid.New().String()
		resp := s.sendRequest(ctx, http.MethodDelete, url+deleteWalletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusNotFound, resp.StatusCode)
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

func (s *IntegrationTestSuite) TestWalletHistory() {
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

		_ = s.sendRequest(ctx, http.MethodPatch, url+updateWalletEndpoint+walletIdEndpoint, reqUpdate, nil)

		_ = s.sendRequest(ctx, http.MethodDelete, url+deleteWalletEndpoint+walletIdEndpoint, nil, nil)

		claimUUID := respData.WalletID.String()
		claimEmail := "go-dev@mail.go"

		var respDataHistory []models.ResponseWalletHistory

		queryParams := fmt.Sprintf("?periodStart=%s&periodEnd=%s",
			time.Now().Add(-12*time.Hour).Format("2006-01-02T15:04:05"),
			time.Now().Format("2006-01-02T15:04:05"))

		resp := s.sendRequestWithCustomClaims(ctx, http.MethodGet, url+walletHistoryEndpoint+queryParams, claimUUID, claimEmail, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(5, len(respDataHistory))

		queryParams = "?textFilter=Noname"
		resp = s.sendRequestWithCustomClaims(ctx, http.MethodGet, url+walletHistoryEndpoint+queryParams, claimUUID, claimEmail, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(2, len(respDataHistory))
		s.Require().Equal("Noname", respDataHistory[0].Owner)

		queryParams = "?sorting=balance&descending=true"
		resp = s.sendRequestWithCustomClaims(ctx, http.MethodGet, url+walletHistoryEndpoint+queryParams, claimUUID, claimEmail, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(float32(1000), respDataHistory[0].Balance)

		queryParams = "?itemsPerPage=3"
		resp = s.sendRequestWithCustomClaims(ctx, http.MethodGet, url+walletHistoryEndpoint+queryParams, claimUUID, claimEmail, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal(3, len(respDataHistory))

		queryParams = "?offset=2"
		resp = s.sendRequestWithCustomClaims(ctx, http.MethodGet, url+walletHistoryEndpoint+queryParams, claimUUID, claimEmail, nil,
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

		claimUUID := respData.WalletID.String()
		claimEmail := "go-dev@email.go"

		var respDataHistory []models.ResponseWalletHistory

		queryParams := fmt.Sprintf("?periodStart=%s&periodEnd=%s",
			time.Now().Format("2006-01-02T15:04:05"),
			time.Now().Add(3*time.Second).Format("2006-01-02T15:04:05"))

		resp := s.sendRequestWithCustomClaims(ctx, http.MethodGet, url+walletHistoryEndpoint+queryParams, claimUUID, claimEmail, nil,
			&respDataHistory)

		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().Equal([]models.ResponseWalletHistory{}, respDataHistory)
	})
}

func (s *IntegrationTestSuite) TestAuthorization() {
	s.Run("request authorization error", func() {
		ctx := context.Background()

		req := models.RequestWalletInstance{}
		req.TransactionKey = uuid.New()
		req.Owner = "Kate"
		req.Currency = "RUB"

		var respData models.ResponseWalletInstance

		_ = s.sendRequest(ctx, http.MethodPost, url+createWalletEndpoint, req, &respData)

		walletIdEndpoint := respData.WalletID.String()
		resp := s.sendRequestWithInvalidToken(ctx, http.MethodGet, url+walletEndpoint+walletIdEndpoint, nil, nil)

		s.Require().Equal(http.StatusUnauthorized, resp.StatusCode)
	})
}
