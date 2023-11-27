package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/AlexZav1327/service/internal/messages"
	"github.com/AlexZav1327/service/internal/notifications"
	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/rates"
	walletserver "github.com/AlexZav1327/service/internal/wallet-server"
	walletservice "github.com/AlexZav1327/service/internal/wallet-service"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

const (
	port                  = 5005
	host                  = ""
	dsn                   = "user=user password=secret host=localhost port=5432 dbname=postgres sslmode=disable"
	createWalletEndpoint  = "/api/v1/wallet/create"
	walletHistoryEndpoint = "/api/v1/wallet/history"
	walletEndpoint        = "/api/v1/wallet/"
	walletsEndpoint       = "/api/v1/wallets"
	updateWalletEndpoint  = "/api/v1/wallet/update/"
	deleteWalletEndpoint  = "/api/v1/wallet/delete/"
	deposit               = "/deposit"
	withdraw              = "/withdraw"
	transfer              = "/transfer/"
)

var url = fmt.Sprintf("http://localhost:%d", port)

type IntegrationTestSuite struct {
	suite.Suite
	pg            *postgres.Postgres
	server        *walletserver.Server
	walletService *walletservice.Service
	xr            *rates.Rates
	message       *messages.Message
	notifications *notifications.Notifications
}

func (s *IntegrationTestSuite) SetupSuite() {
	var (
		signingKey      = os.Getenv("PRIVATE_SIGNING_KEY")
		verificationKey = os.Getenv("PUBLIC_VERIFICATION_KEY")
	)

	ctx := context.Background()
	logger := logrus.StandardLogger()

	var err error

	s.pg, err = postgres.ConnectDB(ctx, logger, dsn)
	s.Require().NoError(err)

	err = s.pg.Migrate(migrate.Up)
	s.Require().NoError(err)

	s.xr = rates.New(logger)
	s.message = messages.New(logger)
	s.notifications = notifications.New(logger)

	s.walletService = walletservice.New(s.pg, s.xr, s.message, s.notifications, logger)

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(signingKey))
	s.Require().NoError(err)

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(verificationKey))
	s.Require().NoError(err)

	s.server = walletserver.New(host, port, s.walletService, logger, privateKey, publicKey)

	go func() {
		_ = s.server.Run(ctx)
	}()

	time.Sleep(250 * time.Millisecond)
}

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

func (s *IntegrationTestSuite) sendRequest(ctx context.Context, method, endpoint string, body,
	dest interface{},
) *http.Response {
	s.T().Helper()

	reqBody, err := json.Marshal(body)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(reqBody))
	s.Require().NoError(err)

	token, err := s.server.GenerateToken("", "")
	s.Require().NoError(err)

	bearer := fmt.Sprintf("Bearer %s", token)

	req.Header.Set("Authorization", bearer)
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

func (s *IntegrationTestSuite) sendRequestWithCustomClaims(ctx context.Context, method, endpoint, claimUUID,
	claimEmail string, body, dest interface{},
) *http.Response {
	s.T().Helper()

	reqBody, err := json.Marshal(body)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(reqBody))
	s.Require().NoError(err)

	token, err := s.server.GenerateToken(claimUUID, claimEmail)
	s.Require().NoError(err)

	bearer := fmt.Sprintf("Bearer %s", token)

	req.Header.Set("Authorization", bearer)
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

func (s *IntegrationTestSuite) sendRequestWithInvalidToken(ctx context.Context, method, endpoint string, body,
	dest interface{},
) *http.Response {
	s.T().Helper()

	reqBody, err := json.Marshal(body)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(reqBody))
	s.Require().NoError(err)

	token := ""
	bearer := fmt.Sprintf("Bearer %s", token)

	req.Header.Set("Authorization", bearer)
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
