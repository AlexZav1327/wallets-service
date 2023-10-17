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
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

const (
	port                  = 5005
	createWalletEndpoint  = "/api/v1/wallet/create"
	walletHistoryEndpoint = "/api/v1/wallet/history"
	walletEndpoint        = "/api/v1/wallet/"
	walletsEndpoint       = "/api/v1/wallets"
	dsn                   = "user=user password=1234 host=localhost port=5432 dbname=postgres sslmode=disable"
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

	token, err := walletserver.GenerateToken("", "")
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

func (s *IntegrationTestSuite) sendRequestWithCustomClaims(ctx context.Context, method, endpoint, claimUUID, claimEmail string, body interface{},
	dest interface{},
) *http.Response {
	s.T().Helper()

	reqBody, err := json.Marshal(body)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(reqBody))
	s.Require().NoError(err)

	token, err := walletserver.GenerateToken(claimUUID, claimEmail)
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

func (s *IntegrationTestSuite) sendRequestWithInvalidToken(ctx context.Context, method, endpoint string, body interface{},
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
