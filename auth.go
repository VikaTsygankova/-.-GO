package middleware

import (
	"bank-service/internal/utils"
	"context"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

type ctxKey string

const UserIDKey ctxKey = "userID"

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" || !strings.HasPrefix(h, "Bearer ") {
				utils.Error(w, http.StatusUnauthorized, "Authorization header required")
				return
			}
			tokenString := strings.TrimPrefix(h, "Bearer ")
			claims := &jwt.RegisteredClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) { return []byte(secret), nil })
			if err != nil || !token.Valid {
				utils.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
