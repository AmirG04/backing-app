package models

import "time"

type User struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	PasswordHash     string    `json:"-"`
	FullName         string    `json:"full_name"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	CreatedAt        time.Time `json:"created_at"`
}

// Account representa una cuenta bancaria de TigerBeetle asociada a un
// usuario. Un usuario puede tener varias cuentas (ej. ahorro y corriente).
type Account struct {
	ID                    string `json:"id"`
	TigerBeetleAccountID  string `json:"tigerbeetle_account_id"`
	AccountNumber         string `json:"account_number,omitempty"`
	AccountType           string `json:"account_type"`
	Currency              string `json:"currency"`
	Balance               *int64 `json:"balance,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

// CreateAccountRequest son los datos opcionales al crear una cuenta
// adicional. Si se omiten, se usa checking/USD por defecto.
type CreateAccountRequest struct {
	AccountType string `json:"account_type"`
	Currency    string `json:"currency"`
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
	Token   string  `json:"token"`
	User    User    `json:"user"`
	Account Account `json:"account"`
}

// LoginResponse es lo que devuelve /api/auth/login. Si el usuario tiene
// 2FA activo, no trae token de sesion todavia - trae un token temporal
// de pre-autenticacion (PreAuthToken) que solo sirve para completar el
// 2FA en /api/auth/2fa/login.
type LoginResponse struct {
	RequiresTwoFactor bool    `json:"requires_2fa"`
	PreAuthToken      string  `json:"pre_auth_token,omitempty"`
	Token             string  `json:"token,omitempty"`
	User              *User   `json:"user,omitempty"`
	Account           *Account `json:"account,omitempty"`
}

type Setup2FAResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

type Verify2FARequest struct {
	Code string `json:"code"`
}

type Login2FARequest struct {
	Code string `json:"code"`
}

type Disable2FARequest struct {
	Password string `json:"password"`
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
	AccountNumber string `json:"account_number"`
	Balance       int64  `json:"balance"`
}

// TransactionHistoryItem es una version legible de una transferencia de
// TigerBeetle, pensada para el frontend (en vez de exponer los bytes
// crudos de Uint128 que usa el SDK internamente).
type TransactionHistoryItem struct {
	ID                        string `json:"id"`
	Type                      string `json:"type"` // deposit | withdrawal | transfer_sent | transfer_received
	Amount                    int64  `json:"amount"`
	CounterpartyAccountNumber string `json:"counterparty_account_number,omitempty"`
	Timestamp                 string `json:"timestamp"` // RFC3339
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
