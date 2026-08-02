package handlers

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"banking-app/backend/internal/ai"
	"banking-app/backend/internal/db"
	"banking-app/backend/internal/models"
)

// pendingAction guarda una operacion critica (retiro/transferencia) que
// el modelo de IA decidio invocar pero que todavia no se ha ejecutado
// porque espera confirmacion explicita del usuario.
type pendingAction struct {
	ToolName string
	Args     map[string]any
}

type Handler struct {
	PG        *pgxpool.Pool
	TB        *db.TigerBeetleClient
	JWTSecret string
	AI        *ai.Client

	pendingMu     sync.Mutex
	pendingByUser map[string]pendingAction
}

func New(pg *pgxpool.Pool, tb *db.TigerBeetleClient, jwtSecret string, aiClient *ai.Client) *Handler {
	return &Handler{
		PG:            pg,
		TB:            tb,
		JWTSecret:     jwtSecret,
		AI:            aiClient,
		pendingByUser: make(map[string]pendingAction),
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Error: message})
}
