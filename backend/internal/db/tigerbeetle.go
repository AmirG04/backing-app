package db

import (
	"fmt"
	"math/big"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// Ledger y Code son valores que tu defines para tu dominio.
// Ledger=1 -> "USD" (o la moneda unica que uses en la prueba)
// Code=1   -> "cuenta de usuario"
// Code=2   -> "cuenta del banco" (origen de depositos, destino de retiros)
const (
	LedgerUSD          = 1
	CodeUserAccount    = 1
	CodeBankAccount    = 2
	BankAccountID      = 1 // ID fijo y reservado para la cuenta "banco"
)

type TigerBeetleClient struct {
	client tb.Client
}

func NewTigerBeetleClient(clusterID uint64, addresses []string) (*TigerBeetleClient, error) {
	client, err := tb.NewClient(tbtypes.ToUint128(clusterID), addresses)
	if err != nil {
		return nil, fmt.Errorf("error creando cliente tigerbeetle: %w", err)
	}
	return &TigerBeetleClient{client: client}, nil
}

func (t *TigerBeetleClient) Close() {
	t.client.Close()
}

// EnsureBankAccount crea la cuenta "banco" (ID=1) si no existe todavia.
// Esta cuenta actua como contraparte para depositos y retiros:
// - Deposito: transfer de BankAccount -> UserAccount
// - Retiro:   transfer de UserAccount -> BankAccount
func (t *TigerBeetleClient) EnsureBankAccount() error {
	id := tbtypes.ToUint128(BankAccountID)
	existing, err := t.client.LookupAccounts([]tbtypes.Uint128{id})
	if err != nil {
		return fmt.Errorf("error buscando cuenta del banco: %w", err)
	}
	if len(existing) > 0 {
		return nil // ya existe
	}

	account := tbtypes.Account{
		ID:     id,
		Ledger: LedgerUSD,
		Code:   CodeBankAccount,
		// El banco puede quedar en debitos negativos (emite dinero),
		// por eso no le ponemos flag de "credits must not exceed debits".
	}

	results, err := t.client.CreateAccounts([]tbtypes.Account{account})
	if err != nil {
		return fmt.Errorf("error creando cuenta del banco: %w", err)
	}
	if len(results) > 0 {
		return fmt.Errorf("error creando cuenta del banco: %s", results[0].Result)
	}
	return nil
}

// CreateUserAccount crea una cuenta TigerBeetle nueva para un usuario.
// Retorna el ID generado (recomendado: usar tbtypes.ID() para IDs
// ordenables en el tiempo, en vez de UUIDs random).
func (t *TigerBeetleClient) CreateUserAccount() (tbtypes.Uint128, error) {
	id := tbtypes.ID() // ID monotono recomendado por TigerBeetle

	account := tbtypes.Account{
		ID:     id,
		Ledger: LedgerUSD,
		Code:   CodeUserAccount,
		Flags: tbtypes.AccountFlags{
			// Evita que el usuario quede con saldo negativo por retiros/transferencias.
			CreditsMustNotExceedDebits: false,
			DebitsMustNotExceedCredits: true,
		}.ToUint16(),
	}

	results, err := t.client.CreateAccounts([]tbtypes.Account{account})
	if err != nil {
		return id, fmt.Errorf("error creando cuenta de usuario: %w", err)
	}
	if len(results) > 0 {
		return id, fmt.Errorf("error creando cuenta de usuario: %s", results[0].Result)
	}
	return id, nil
}

// GetBalance retorna (saldo disponible) de una cuenta.
// En el modelo contable de TigerBeetle, para una cuenta con
// DebitsMustNotExceedCredits, el saldo "disponible" es credits_posted - debits_posted.
func (t *TigerBeetleClient) GetBalance(accountID tbtypes.Uint128) (int64, error) {
	accounts, err := t.client.LookupAccounts([]tbtypes.Uint128{accountID})
	if err != nil {
		return 0, fmt.Errorf("error consultando cuenta: %w", err)
	}
	if len(accounts) == 0 {
		return 0, fmt.Errorf("cuenta no encontrada")
	}
	acc := accounts[0]
	creditsPosted := acc.CreditsPosted.BigInt()
	debitsPosted := acc.DebitsPosted.BigInt()
	balance := new(big.Int).Sub(&creditsPosted, &debitsPosted)
	return balance.Int64(), nil
}

// Deposit agrega fondos: banco -> usuario
func (t *TigerBeetleClient) Deposit(userAccountID tbtypes.Uint128, amount uint64) error {
	return t.transfer(tbtypes.ToUint128(BankAccountID), userAccountID, amount, CodeUserAccount)
}

// Withdraw retira fondos: usuario -> banco
// TigerBeetle valida automaticamente fondos insuficientes gracias al flag
// DebitsMustNotExceedCredits configurado en la cuenta del usuario.
func (t *TigerBeetleClient) Withdraw(userAccountID tbtypes.Uint128, amount uint64) error {
	return t.transfer(userAccountID, tbtypes.ToUint128(BankAccountID), amount, CodeUserAccount)
}

// Transfer mueve fondos entre dos cuentas de usuario.
func (t *TigerBeetleClient) Transfer(fromAccountID, toAccountID tbtypes.Uint128, amount uint64) error {
	return t.transfer(fromAccountID, toAccountID, amount, CodeUserAccount)
}

func (t *TigerBeetleClient) transfer(debitAccountID, creditAccountID tbtypes.Uint128, amount uint64, code uint16) error {
	transfer := tbtypes.Transfer{
		ID:              tbtypes.ID(),
		DebitAccountID:  debitAccountID,
		CreditAccountID: creditAccountID,
		Amount:          tbtypes.ToUint128(amount),
		Ledger:          LedgerUSD,
		Code:            code,
	}

	results, err := t.client.CreateTransfers([]tbtypes.Transfer{transfer})
	if err != nil {
		return fmt.Errorf("error creando transferencia: %w", err)
	}
	if len(results) > 0 {
		// Aqui es donde TigerBeetle rechaza, por ejemplo, fondos insuficientes
		// (exceeds_credits) o cuenta destino inexistente.
		return fmt.Errorf("transferencia rechazada: %s", results[0].Result)
	}
	return nil
}

// GetHistory retorna las transferencias (depositos, retiros, envios y
// recepciones) asociadas a una cuenta, mas recientes primero.
func (t *TigerBeetleClient) GetHistory(accountID tbtypes.Uint128, limit uint32) ([]tbtypes.Transfer, error) {
	filter := tbtypes.AccountFilter{
		AccountID: accountID,
		Limit:     limit,
		Flags: tbtypes.AccountFilterFlags{
			Debits:  true,
			Credits: true,
			Reversed: true,
		}.ToUint32(),
	}
	transfers, err := t.client.GetAccountTransfers(filter)
	if err != nil {
		return nil, fmt.Errorf("error consultando historial: %w", err)
	}
	return transfers, nil
}
