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

type ErrorResponse struct {
	Error string `json:"error"`
}
