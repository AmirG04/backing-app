package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/db"
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
		email       string
		fullName    string
		createdAt   time.Time
		totpEnabled bool
	)
	err := h.PG.QueryRow(r.Context(),
		`SELECT email, full_name, created_at, totp_enabled FROM users WHERE id = $1`, userID,
	).Scan(&email, &fullName, &createdAt, &totpEnabled)
	if err != nil {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}

	writeJSON(w, http.StatusOK, models.User{
		ID:               userID,
		Email:            email,
		FullName:         fullName,
		CreatedAt:        createdAt,
		TwoFactorEnabled: totpEnabled,
	})
}

var validAccountTypes = map[string]bool{"checking": true, "savings": true}

// POST /api/accounts
// Crea una cuenta bancaria adicional para el usuario autenticado (un
// usuario ya puede tener una - creada al registrarse - y puede crear
// mas, ej. una de ahorro ademas de la corriente).
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req models.CreateAccountRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body es opcional, defaults abajo

	accountType := strings.ToLower(strings.TrimSpace(req.AccountType))
	if accountType == "" {
		accountType = "checking"
	}
	if !validAccountTypes[accountType] {
		writeError(w, http.StatusBadRequest, "account_type debe ser 'checking' o 'savings'")
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "USD"
	}

	tbAccountID, err := h.TB.CreateUserAccount()
	if err != nil {
		log.Printf("error creando cuenta tigerbeetle: %v", err)
		writeError(w, http.StatusInternalServerError, "error creando cuenta bancaria")
		return
	}

	var accountID, accountNumber string
	var createdAt time.Time
	const maxAttempts = 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		accountNumber, err = db.GenerateAccountNumber()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "error generando numero de cuenta")
			return
		}

		err = h.PG.QueryRow(r.Context(),
			`INSERT INTO accounts (user_id, tigerbeetle_account_id, account_number, account_type, currency)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
			userID, tbAccountID.String(), accountNumber, accountType, currency,
		).Scan(&accountID, &createdAt)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "duplicate key") || attempt == maxAttempts {
			log.Printf("error creando cuenta en postgres: %v", err)
			writeError(w, http.StatusInternalServerError, "error creando cuenta")
			return
		}
		// numero de cuenta duplicado (muy raro) - reintenta con uno nuevo
	}

	writeJSON(w, http.StatusCreated, models.Account{
		ID:                   accountID,
		TigerBeetleAccountID: tbAccountID.String(),
		AccountNumber:        accountNumber,
		AccountType:          accountType,
		Currency:             currency,
		CreatedAt:            createdAt,
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
