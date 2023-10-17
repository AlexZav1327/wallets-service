package tests

import (
	"context"
	"net/http"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/google/uuid"
)

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
