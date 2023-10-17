package walletserver

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidSigningMethod = errors.New("invalid signing method")
)

type Claims struct {
	jwt.RegisteredClaims
	UUID  string
	Email string
}

func GenerateToken(uuid, email string) (string, error) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UUID:  uuid,
		Email: email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	privateKey, err := getPrivateKey()
	if err != nil {
		return "", fmt.Errorf("GetPrivateKey: %w", err)
	}

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("token.SignedString: %w", err)
	}

	return tokenString, nil
}

func (*Handler) JwtAuth(next http.Handler) http.Handler {
	var fn http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if headerParts[0] != "Bearer" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		sessionInfo, err := verifyToken(headerParts[1])
		if errors.Is(err, ErrInvalidToken) {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		r = r.WithContext(context.WithValue(r.Context(), models.SessionInfo{}, sessionInfo))
		next.ServeHTTP(w, r)
	}

	return fn
}

func verifyToken(accessToken string) (models.SessionInfo, error) {
	publicKey, err := getPublicKey()
	if err != nil {
		return models.SessionInfo{}, fmt.Errorf("getPublicKey: %w", err)
	}

	var sessionInfo models.SessionInfo

	token, err := jwt.ParseWithClaims(accessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodRSA)
		if !ok {
			return nil, ErrInvalidSigningMethod
		}

		return publicKey, nil
	})
	if err != nil {
		return models.SessionInfo{}, fmt.Errorf("jwt.ParseWithClaims: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if ok && token.Valid {
		sessionInfo.UUID = claims.UUID
		sessionInfo.Email = claims.Email

		return sessionInfo, nil
	}

	return models.SessionInfo{}, ErrInvalidToken
}

func getPrivateKey() (*rsa.PrivateKey, error) {
	signingKey := os.Getenv("PRIVATE_SIGNING_KEY")

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(signingKey))
	if err != nil {
		return nil, fmt.Errorf("jwt.ParseRSAPrivateKeyFromPEM: %w", err)
	}

	return privateKey, nil
}

func getPublicKey() (*rsa.PublicKey, error) {
	verificationKey := os.Getenv("PUBLIC_VERIFICATION_KEY")

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(verificationKey))
	if err != nil {
		return nil, fmt.Errorf("jwt.ParseRSAPublicKeyFromPEM: %w", err)
	}

	return publicKey, nil
}
