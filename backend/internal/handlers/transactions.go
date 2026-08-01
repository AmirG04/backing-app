package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/models"
)

// POST /api/transactions/deposit
func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	var req models.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount == 0 {
		writeError(w, http.StatusBadRequest, "monto invalido")
		return
	}

	accountID, _, err := h.getUserTBAccountID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "cuenta no encontrada")
		return
	}

	if err := h.TB.Deposit(accountID, req.Amount); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "no se pudo procesar el deposito: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deposito realizado"})
}

// POST /api/transactions/withdraw
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var req models.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount == 0 {
		writeError(w, http.StatusBadRequest, "monto invalido")
		return
	}

	accountID, _, err := h.getUserTBAccountID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "cuenta no encontrada")
		return
	}

	if err := h.TB.Withdraw(accountID, req.Amount); err != nil {
		// TigerBeetle rechaza automaticamente si no hay fondos suficientes
		// (result: exceeds_credits), gracias al flag DebitsMustNotExceedCredits.
		if strings.Contains(err.Error(), "exceeds_credits") {
			writeError(w, http.StatusUnprocessableEntity, "fondos insuficientes")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "no se pudo procesar el retiro: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "retiro realizado"})
}

// POST /api/transactions/transfer
func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req models.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount == 0 || req.ToAccountID == "" {
		writeError(w, http.StatusBadRequest, "datos invalidos: to_account_id y amount son requeridos")
		return
	}

	fromAccountID, _, err := h.getUserTBAccountID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "cuenta origen no encontrada")
		return
	}

	toAccountID, err := tbtypes.HexStringToUint128(req.ToAccountID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "identificador de cuenta destino invalido")
		return
	}

	if toAccountID == fromAccountID {
		writeError(w, http.StatusBadRequest, "no puedes transferir a tu propia cuenta")
		return
	}

	// Validar que la cuenta destino exista antes de intentar la transferencia
	var exists bool
	err = h.PG.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM users WHERE tigerbeetle_account_id = $1)`,
		req.ToAccountID,
	).Scan(&exists)
	if err != nil || !exists {
		writeError(w, http.StatusNotFound, "la cuenta destino no existe")
		return
	}

	if err := h.TB.Transfer(fromAccountID, toAccountID, req.Amount); err != nil {
		if strings.Contains(err.Error(), "exceeds_credits") {
			writeError(w, http.StatusUnprocessableEntity, "fondos insuficientes")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "no se pudo procesar la transferencia: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "transferencia realizada"})
}

// GET /api/transactions/history
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	accountID, _, err := h.getUserTBAccountID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "cuenta no encontrada")
		return
	}

	transfers, err := h.TB.GetHistory(accountID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error consultando historial")
		return
	}

	writeJSON(w, http.StatusOK, transfers)
}
