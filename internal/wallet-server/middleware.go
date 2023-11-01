package walletserver

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AlexZav1327/service/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

func (h *Handler) generateToken(uuid, email string) (string, error) {
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

	tokenString, err := token.SignedString(h.privateKey)
	if err != nil {
		return "", fmt.Errorf("token.SignedString: %w", err)
	}

	return tokenString, nil
}

func (h *Handler) verifyToken(accessToken string, publicKey *rsa.PublicKey) (models.SessionInfo, error) {
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

func (h *Handler) jwtAuth(next http.Handler) http.Handler {
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

		sessionInfo, err := h.verifyToken(headerParts[1], h.publicKey)
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

func (h *Handler) metric(next http.Handler) http.Handler {
	var fn http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		pattern := chi.RouteContext(r.Context()).RoutePattern()

		h.metrics.duration.WithLabelValues(http.StatusText(ww.Status()), r.Method,
			pattern).Observe(time.Since(started).Seconds())
		h.metrics.requests.WithLabelValues(http.StatusText(ww.Status()), r.Method, pattern).Inc()
	}

	return fn
}
