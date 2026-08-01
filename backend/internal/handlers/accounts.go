package handlers

import (
	"net/http"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/middleware"
	"banking-app/backend/internal/models"
)

// getUserTBAccountID resuelve el ID de cuenta TigerBeetle del usuario autenticado.
func (h *Handler) getUserTBAccountID(r *http.Request) (tbtypes.Uint128, string, error) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var tbAccountIDStr string
	err := h.PG.QueryRow(r.Context(),
		`SELECT tigerbeetle_account_id FROM users WHERE id = $1`, userID,
	).Scan(&tbAccountIDStr)
	if err != nil {
		return tbtypes.Uint128{}, "", err
	}

	id, err := tbtypes.HexStringToUint128(tbAccountIDStr)
	if err != nil {
		return tbtypes.Uint128{}, "", err
	}
	return id, tbAccountIDStr, nil
}

// GET /api/accounts/me
func (h *Handler) GetAccountInfo(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var (
		email       string
		fullName    string
		tbAccountID string
	)
	err := h.PG.QueryRow(r.Context(),
		`SELECT email, full_name, tigerbeetle_account_id FROM users WHERE id = $1`, userID,
	).Scan(&email, &fullName, &tbAccountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	writeJSON(w, http.StatusOK, models.User{
		ID:                   userID,
		Email:                email,
		FullName:             fullName,
		TigerBeetleAccountID: tbAccountID,
	})
}

// GET /api/accounts/balance
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	accountID, accountIDStr, err := h.getUserTBAccountID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "cuenta no encontrada")
		return
	}

	balance, err := h.TB.GetBalance(accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error consultando saldo")
		return
	}

	writeJSON(w, http.StatusOK, models.BalanceResponse{
		AccountID: accountIDStr,
		Balance:   balance,
	})
}
