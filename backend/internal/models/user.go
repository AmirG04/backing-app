package models

import "time"

type User struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	PasswordHash         string    `json:"-"`
	FullName             string    `json:"full_name"`
	TigerBeetleAccountID string    `json:"tigerbeetle_account_id"`
	CreatedAt            time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type DepositRequest struct {
	Amount uint64 `json:"amount"`
}

type WithdrawRequest struct {
	Amount uint64 `json:"amount"`
}

type TransferRequest struct {
	ToAccountID string `json:"to_account_id"`
	Amount      uint64 `json:"amount"`
}

type BalanceResponse struct {
	AccountID string `json:"account_id"`
	Balance   int64  `json:"balance"`
}

// TransactionHistoryItem es una version legible de una transferencia de
// TigerBeetle, pensada para el frontend (en vez de exponer los bytes
// crudos de Uint128 que usa el SDK internamente).
type TransactionHistoryItem struct {
	ID                    string `json:"id"`
	Type                  string `json:"type"` // deposit | withdrawal | transfer_sent | transfer_received
	Amount                int64  `json:"amount"`
	CounterpartyAccountID string `json:"counterparty_account_id,omitempty"`
	Timestamp             string `json:"timestamp"` // RFC3339
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply             string `json:"reply"`
	ActionExecuted    bool   `json:"action_executed"`
	RequiresConfirmation bool `json:"requires_confirmation"`
}
