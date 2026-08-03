package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDContextKey contextKey = "user_id"

// Auth valida el JWT del header Authorization: Bearer <token>
// e inyecta el user_id en el contexto de la request.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"token no proporcionado"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"token invalido o expirado"}`, http.StatusUnauthorized)
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				http.Error(w, `{"error":"token invalido"}`, http.StatusUnauthorized)
				return
			}

			// Los tokens de pre-autenticacion (emitidos mientras el usuario
			// todavia no completa el 2FA) llevan una marca "purpose" y NUNCA
			// deben aceptarse como sesion valida en rutas protegidas - solo
			// sirven para el endpoint /api/auth/2fa/login.
			if _, hasPurpose := claims["purpose"]; hasPurpose {
				http.Error(w, `{"error":"token invalido"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}
