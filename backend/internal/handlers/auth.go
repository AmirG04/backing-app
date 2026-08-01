package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"banking-app/backend/internal/models"
)

// POST /api/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la peticion invalido")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		writeError(w, http.StatusBadRequest, "email, password y full_name son requeridos")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "la contraseña debe tener al menos 8 caracteres")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error procesando la contraseña")
		return
	}

	// 1. Crear cuenta financiera en TigerBeetle
	tbAccountID, err := h.TB.CreateUserAccount()
	if err != nil {
		log.Printf("error creando cuenta tigerbeetle: %v", err)
		writeError(w, http.StatusInternalServerError, "error creando cuenta bancaria")
		return
	}

	// 2. Crear usuario en Postgres, referenciando la cuenta de TigerBeetle
	var userID string
	err = h.PG.QueryRow(r.Context(),
		`INSERT INTO users (email, password_hash, full_name, tigerbeetle_account_id)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		req.Email, string(hash), req.FullName, tbAccountID.String(),
	).Scan(&userID)
	if err != nil {
		// Nota: si esto falla, la cuenta en TigerBeetle ya quedo creada
		// (no hay rollback automatico entre las dos bases). Para una prueba
		// tecnica esto es aceptable; en produccion se usaria un patron saga
		// o outbox para mantener consistencia entre ambos sistemas.
		if strings.Contains(err.Error(), "duplicate key") {
			writeError(w, http.StatusConflict, "el email ya esta registrado")
			return
		}
		log.Printf("error creando usuario en postgres: %v", err)
		writeError(w, http.StatusInternalServerError, "error creando usuario")
		return
	}

	token, err := h.generateToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error generando token")
		return
	}

	writeJSON(w, http.StatusCreated, models.AuthResponse{
		Token: token,
		User: models.User{
			ID:                   userID,
			Email:                req.Email,
			FullName:             req.FullName,
			TigerBeetleAccountID: tbAccountID.String(),
		},
	})
}

// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo de la peticion invalido")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var (
		userID       string
		passwordHash string
		fullName     string
		tbAccountID  string
	)
	err := h.PG.QueryRow(r.Context(),
		`SELECT id, password_hash, full_name, tigerbeetle_account_id
		 FROM users WHERE email = $1`, req.Email,
	).Scan(&userID, &passwordHash, &fullName, &tbAccountID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "credenciales invalidas")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "credenciales invalidas")
		return
	}

	token, err := h.generateToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error generando token")
		return
	}

	writeJSON(w, http.StatusOK, models.AuthResponse{
		Token: token,
		User: models.User{
			ID:                   userID,
			Email:                req.Email,
			FullName:             fullName,
			TigerBeetleAccountID: tbAccountID,
		},
	})
}

// POST /api/auth/logout
// Con JWT sin estado, el logout real ocurre del lado del cliente
// (borrando el token). Este endpoint existe por completitud de la API
// y como punto de extension si luego se agrega una blacklist de tokens.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "sesion cerrada"})
}

func (h *Handler) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.JWTSecret))
}
