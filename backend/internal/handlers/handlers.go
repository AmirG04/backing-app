package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"banking-app/backend/internal/db"
	"banking-app/backend/internal/models"
)

type Handler struct {
	PG        *pgxpool.Pool
	TB        *db.TigerBeetleClient
	JWTSecret string
}

func New(pg *pgxpool.Pool, tb *db.TigerBeetleClient, jwtSecret string) *Handler {
	return &Handler{PG: pg, TB: tb, JWTSecret: jwtSecret}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Error: message})
}
