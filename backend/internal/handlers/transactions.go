package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/db"
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
		// (result: ExceedsCredits), gracias al flag DebitsMustNotExceedCredits.
		if strings.Contains(err.Error(), "ExceedsCredits") {
			writeError(w, http.StatusUnprocessableEntity, "fondos insuficientes")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "no se pudo procesar el retiro: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "retiro realizado"})
}

// POST /api/transactions/transfer
// req.ToAccountID es el NUMERO DE CUENTA publico (ej. "4001-1234-5678-0001"),
// no el ID interno de TigerBeetle - se resuelve el uno al otro consultando Postgres.
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

	var toTBHex string
	err = h.PG.QueryRow(r.Context(),
		`SELECT tigerbeetle_account_id FROM accounts WHERE account_number = $1`,
		req.ToAccountID,
	).Scan(&toTBHex)
	if err != nil {
		writeError(w, http.StatusNotFound, "la cuenta destino no existe")
		return
	}

	toAccountID, err := tbtypes.HexStringToUint128(toTBHex)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cuenta destino invalida")
		return
	}

	if toAccountID == fromAccountID {
		writeError(w, http.StatusBadRequest, "no puedes transferir a tu propia cuenta")
		return
	}

	if err := h.TB.Transfer(fromAccountID, toAccountID, req.Amount); err != nil {
		if strings.Contains(err.Error(), "ExceedsCredits") {
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

	bankAccountID := tbtypes.ToUint128(db.BankAccountID)

	items := make([]models.TransactionHistoryItem, 0, len(transfers))
	for _, t := range transfers {
		amountBig := t.Amount.BigInt()

		item := models.TransactionHistoryItem{
			ID:        t.ID.String(),
			Amount:    amountBig.Int64(),
			Timestamp: time.Unix(0, int64(t.Timestamp)).UTC().Format(time.RFC3339),
		}

		debitIsBank := t.DebitAccountID == bankAccountID
		creditIsBank := t.CreditAccountID == bankAccountID

		var counterpartyTBHex string
		switch {
		case debitIsBank && !creditIsBank:
			item.Type = "deposit"
		case !debitIsBank && creditIsBank:
			item.Type = "withdrawal"
		case t.DebitAccountID == accountID:
			item.Type = "transfer_sent"
			counterpartyTBHex = t.CreditAccountID.String()
		case t.CreditAccountID == accountID:
			item.Type = "transfer_received"
			counterpartyTBHex = t.DebitAccountID.String()
		default:
			item.Type = "unknown"
		}

		if counterpartyTBHex != "" {
			var accountNumber string
			err := h.PG.QueryRow(r.Context(),
				`SELECT account_number FROM accounts WHERE tigerbeetle_account_id = $1`,
				counterpartyTBHex,
			).Scan(&accountNumber)
			if err == nil {
				item.CounterpartyAccountNumber = accountNumber
			}
		}

		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, items)
}
