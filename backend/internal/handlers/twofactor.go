package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"banking-app/backend/internal/middleware"
	"banking-app/backend/internal/models"
)

// POST /api/2fa/setup
// Genera un secreto TOTP nuevo para el usuario autenticado y lo guarda
// como "pendiente" (totp_enabled sigue en false hasta que se confirme
// con /api/2fa/verify).
func (h *Handler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var email string
	if err := h.PG.QueryRow(r.Context(), `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Banco Simplificado",
		AccountName: email,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error generando el secreto 2FA")
		return
	}

	_, err = h.PG.Exec(r.Context(),
		`UPDATE users SET totp_secret = $1, totp_enabled = false WHERE id = $2`,
		key.Secret(), userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error guardando el secreto 2FA")
		return
	}

	writeJSON(w, http.StatusOK, models.Setup2FAResponse{
		Secret:     key.Secret(),
		OTPAuthURL: key.URL(),
	})
}

// POST /api/2fa/verify
// Confirma la activacion de 2FA: el usuario manda el primer codigo
// generado por su app autenticadora, y si es valido, 2FA queda activo.
func (h *Handler) Verify2FA(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req models.Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		writeError(w, http.StatusBadRequest, "codigo invalido")
		return
	}

	var secret string
	if err := h.PG.QueryRow(r.Context(), `SELECT COALESCE(totp_secret, '') FROM users WHERE id = $1`, userID).Scan(&secret); err != nil || secret == "" {
		writeError(w, http.StatusBadRequest, "primero debes iniciar la configuracion en /api/2fa/setup")
		return
	}

	if !totp.Validate(req.Code, secret) {
		writeError(w, http.StatusUnauthorized, "codigo invalido")
		return
	}

	if _, err := h.PG.Exec(r.Context(), `UPDATE users SET totp_enabled = true WHERE id = $1`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "error activando 2FA")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA activado correctamente"})
}

// POST /api/2fa/disable
// Requiere la contraseña actual (no solo estar logueado) por seguridad,
// ya que desactivar 2FA reduce la proteccion de la cuenta.
func (h *Handler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req models.Disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		writeError(w, http.StatusBadRequest, "se requiere la contraseña actual")
		return
	}

	var passwordHash string
	if err := h.PG.QueryRow(r.Context(), `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&passwordHash); err != nil {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "contraseña incorrecta")
		return
	}

	if _, err := h.PG.Exec(r.Context(), `UPDATE users SET totp_enabled = false, totp_secret = NULL WHERE id = $1`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "error desactivando 2FA")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA desactivado"})
}

// POST /api/auth/2fa/login
// Segundo paso del login cuando el usuario tiene 2FA activo. NO usa el
// middleware de autenticacion normal (ese rechaza explicitamente los
// tokens de pre-autenticacion) - valida el token de pre-autenticacion a
// mano, y solo acepta uno con purpose="2fa_pending".
func (h *Handler) Login2FA(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "token de pre-autenticacion requerido")
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		writeError(w, http.StatusUnauthorized, "token invalido o expirado")
		return
	}
	if purpose, _ := claims["purpose"].(string); purpose != "2fa_pending" {
		writeError(w, http.StatusUnauthorized, "token invalido")
		return
	}
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "token invalido")
		return
	}

	var req models.Login2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		writeError(w, http.StatusBadRequest, "codigo requerido")
		return
	}

	var (
		email, fullName, secret string
	)
	err = h.PG.QueryRow(r.Context(),
		`SELECT email, full_name, COALESCE(totp_secret, '') FROM users WHERE id = $1`, userID,
	).Scan(&email, &fullName, &secret)
	if err != nil || secret == "" {
		writeError(w, http.StatusUnauthorized, "2FA no esta configurado para este usuario")
		return
	}

	if !totp.Validate(req.Code, secret) {
		writeError(w, http.StatusUnauthorized, "codigo invalido")
		return
	}

	var account models.Account
	err = h.PG.QueryRow(r.Context(),
		`SELECT id, tigerbeetle_account_id, COALESCE(account_number, ''), account_type, currency, created_at
		 FROM accounts WHERE user_id = $1 ORDER BY created_at ASC LIMIT 1`, userID,
	).Scan(&account.ID, &account.TigerBeetleAccountID, &account.AccountNumber, &account.AccountType, &account.Currency, &account.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "el usuario no tiene ninguna cuenta asociada")
		return
	}

	sessionToken, err := h.generateToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error generando token")
		return
	}

	user := models.User{ID: userID, Email: email, FullName: fullName}
	writeJSON(w, http.StatusOK, models.LoginResponse{
		Token:   sessionToken,
		User:    &user,
		Account: &account,
	})
}
