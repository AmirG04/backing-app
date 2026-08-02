package handlers

import (
	"net/http"
	"time"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/middleware"
	"banking-app/backend/internal/models"
)

// resolveAccount averigua sobre cual cuenta debe operar la peticion.
// Si viene un ?account_id=<uuid> en la query string, usa esa cuenta
// (validando que le pertenezca al usuario autenticado). Si no viene,
// usa la cuenta "principal" del usuario (la mas antigua). Esto permite
// que un usuario tenga varias cuentas sin romper los endpoints que ya
// existian antes de soportar multiples cuentas.
func (h *Handler) resolveAccount(r *http.Request) (tbtypes.Uint128, models.Account, error) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	requestedID := r.URL.Query().Get("account_id")

	var acc models.Account
	var err error

	if requestedID != "" {
		err = h.PG.QueryRow(r.Context(),
			`SELECT id, tigerbeetle_account_id, COALESCE(account_number, ''), account_type, currency, created_at
			 FROM accounts WHERE id = $1 AND user_id = $2`, requestedID, userID,
		).Scan(&acc.ID, &acc.TigerBeetleAccountID, &acc.AccountNumber, &acc.AccountType, &acc.Currency, &acc.CreatedAt)
	} else {
		err = h.PG.QueryRow(r.Context(),
			`SELECT id, tigerbeetle_account_id, COALESCE(account_number, ''), account_type, currency, created_at
			 FROM accounts WHERE user_id = $1 ORDER BY created_at ASC LIMIT 1`, userID,
		).Scan(&acc.ID, &acc.TigerBeetleAccountID, &acc.AccountNumber, &acc.AccountType, &acc.Currency, &acc.CreatedAt)
	}
	if err != nil {
		return tbtypes.Uint128{}, models.Account{}, err
	}

	tbID, err := tbtypes.HexStringToUint128(acc.TigerBeetleAccountID)
	if err != nil {
		return tbtypes.Uint128{}, models.Account{}, err
	}
	return tbID, acc, nil
}

// getUserTBAccountID es un atajo sobre resolveAccount para el codigo que
// solo necesita el ID de TigerBeetle (no el resto de la info de la cuenta).
func (h *Handler) getUserTBAccountID(r *http.Request) (tbtypes.Uint128, string, error) {
	tbID, acc, err := h.resolveAccount(r)
	if err != nil {
		return tbtypes.Uint128{}, "", err
	}
	return tbID, acc.TigerBeetleAccountID, nil
}

// GET /api/accounts/me
func (h *Handler) GetAccountInfo(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var (
		email     string
		fullName  string
		createdAt time.Time
	)
	err := h.PG.QueryRow(r.Context(),
		`SELECT email, full_name, created_at FROM users WHERE id = $1`, userID,
	).Scan(&email, &fullName, &createdAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	writeJSON(w, http.StatusOK, models.User{
		ID:        userID,
		Email:     email,
		FullName:  fullName,
		CreatedAt: createdAt,
	})
}

// GET /api/accounts
// Lista todas las cuentas del usuario autenticado, cada una con su saldo
// actual (consultado a TigerBeetle).
func (h *Handler) GetAccounts(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	rows, err := h.PG.Query(r.Context(),
		`SELECT id, tigerbeetle_account_id, COALESCE(account_number, ''), account_type, currency, created_at
		 FROM accounts WHERE user_id = $1 ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error consultando cuentas")
		return
	}
	defer rows.Close()

	accounts := []models.Account{}
	for rows.Next() {
		var acc models.Account
		if err := rows.Scan(&acc.ID, &acc.TigerBeetleAccountID, &acc.AccountNumber, &acc.AccountType, &acc.Currency, &acc.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "error leyendo cuentas")
			return
		}

		tbID, err := tbtypes.HexStringToUint128(acc.TigerBeetleAccountID)
		if err == nil {
			if balance, err := h.TB.GetBalance(tbID); err == nil {
				acc.Balance = &balance
			}
		}
		accounts = append(accounts, acc)
	}

	writeJSON(w, http.StatusOK, accounts)
}

// GET /api/accounts/balance
// Acepta ?account_id=<uuid> opcional para consultar una cuenta especifica;
// sin ese parametro, usa la cuenta principal del usuario.
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	tbID, acc, err := h.resolveAccount(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "cuenta no encontrada")
		return
	}

	balance, err := h.TB.GetBalance(tbID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error consultando saldo")
		return
	}

	writeJSON(w, http.StatusOK, models.BalanceResponse{
		AccountNumber: acc.AccountNumber,
		Balance:       balance,
	})
}
