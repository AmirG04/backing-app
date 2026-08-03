// Package mcp implementa un servidor de Model Context Protocol que expone
// las operaciones bancarias (saldo, historial, deposito, retiro,
// transferencia) como herramientas ("tools") que un modelo de IA puede
// descubrir e invocar. Usa el SDK oficial de Go para MCP
// (github.com/modelcontextprotocol/go-sdk).
package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	gosdk "github.com/modelcontextprotocol/go-sdk/mcp"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/db"
)

const (
	ToolListAccounts = "list_accounts"
	ToolGetBalance    = "get_balance"
	ToolGetHistory    = "get_history"
	ToolDeposit       = "deposit"
	ToolWithdraw      = "withdraw"
	ToolTransfer      = "transfer"
)

// CriticalTools son las operaciones que mueven dinero fuera de la cuenta
// del usuario y por lo tanto requieren confirmacion explicita antes de
// ejecutarse (requisito del enunciado: "La IA debe confirmar acciones
// criticas antes de ejecutarlas").
var CriticalTools = map[string]bool{
	ToolWithdraw: true,
	ToolTransfer: true,
}

type emptyInput struct{}

// accountNumberInput se usa en herramientas de solo lectura que pueden
// operar sobre cualquier cuenta del usuario. AccountNumber es opcional:
// si el usuario tiene una sola cuenta, se usa esa; si tiene varias y no
// se especifica, la herramienta responde pidiendo que se especifique.
type accountNumberInput struct {
	AccountNumber string `json:"account_number,omitempty" jsonschema:"numero de cuenta sobre la que operar. Opcional si el usuario solo tiene una cuenta; obligatorio (o preguntar al usuario) si tiene varias."`
}

type amountInput struct {
	AccountNumber string  `json:"account_number,omitempty" jsonschema:"numero de cuenta sobre la que operar. Opcional si el usuario solo tiene una cuenta; obligatorio (o preguntar al usuario) si tiene varias."`
	Amount        float64 `json:"amount" jsonschema:"monto de la operacion, debe ser mayor a 0"`
}

type transferInput struct {
	FromAccountNumber string  `json:"from_account_number,omitempty" jsonschema:"cuenta ORIGEN del usuario. Opcional si el usuario solo tiene una cuenta; obligatorio (o preguntar al usuario) si tiene varias."`
	ToAccountID       string  `json:"to_account_id" jsonschema:"numero de cuenta DESTINO, formato 4001-XXXX-XXXX-NNNN"`
	Amount            float64 `json:"amount" jsonschema:"monto a transferir, debe ser mayor a 0"`
}

func textResult(msg string) *gosdk.CallToolResult {
	return &gosdk.CallToolResult{Content: []gosdk.Content{&gosdk.TextContent{Text: msg}}}
}

func errorResult(msg string) *gosdk.CallToolResult {
	return &gosdk.CallToolResult{IsError: true, Content: []gosdk.Content{&gosdk.TextContent{Text: msg}}}
}

// userAccount es la info minima de una cuenta necesaria para resolver
// sobre cual operar.
type userAccount struct {
	TBID   tbtypes.Uint128
	Number string
	Type   string
}

// loadUserAccounts trae todas las cuentas del usuario autenticado.
func loadUserAccounts(ctx context.Context, pg *pgxpool.Pool, userID string) ([]userAccount, error) {
	rows, err := pg.Query(ctx,
		`SELECT tigerbeetle_account_id, account_number, account_type
		 FROM accounts WHERE user_id = $1 ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []userAccount
	for rows.Next() {
		var tbHex, number, accType string
		if err := rows.Scan(&tbHex, &number, &accType); err != nil {
			return nil, err
		}
		tbID, err := tbtypes.HexStringToUint128(tbHex)
		if err != nil {
			continue
		}
		accounts = append(accounts, userAccount{TBID: tbID, Number: number, Type: accType})
	}
	return accounts, nil
}

// pickAccount decide sobre cual cuenta operar. Si el usuario pidio una
// especifica (requested != ""), la busca entre las suyas. Si no pidio
// ninguna: usa la unica que tenga, o si tiene varias, retorna un
// CallToolResult listando las opciones para que el modelo le pregunte
// al usuario cual quiere usar (en vez de adivinar).
func pickAccount(accounts []userAccount, requested string) (userAccount, *gosdk.CallToolResult) {
	if requested != "" {
		for _, a := range accounts {
			if strings.EqualFold(a.Number, requested) {
				return a, nil
			}
		}
		return userAccount{}, errorResult("no se encontro una cuenta tuya con ese numero")
	}

	switch len(accounts) {
	case 0:
		return userAccount{}, errorResult("el usuario no tiene ninguna cuenta bancaria")
	case 1:
		return accounts[0], nil
	default:
		var sb strings.Builder
		sb.WriteString("El usuario tiene varias cuentas, hay que preguntarle cual usar. Opciones: ")
		for i, a := range accounts {
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(fmt.Sprintf("%s (%s)", a.Number, a.Type))
		}
		return userAccount{}, errorResult(sb.String())
	}
}

// NewBankingServer crea un servidor MCP con las herramientas bancarias
// atadas a un usuario especifico (no a una sola cuenta - un usuario
// puede tener varias). Se crea uno nuevo por cada conversacion de chat,
// de forma que las herramientas solo puedan operar sobre las cuentas del
// usuario autenticado en esa peticion - nunca sobre las de otro usuario.
func NewBankingServer(pg *pgxpool.Pool, tb *db.TigerBeetleClient, userID string) *gosdk.Server {
	server := gosdk.NewServer(&gosdk.Implementation{Name: "banking-mcp-server", Version: "v1.0.0"}, nil)

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolListAccounts,
		Description: "Lista todas las cuentas bancarias del usuario autenticado, con su tipo, numero y saldo. Usala cuando el usuario pregunte cuantas cuentas tiene o pida verlas todas.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, _ emptyInput) (*gosdk.CallToolResult, any, error) {
		accounts, err := loadUserAccounts(ctx, pg, userID)
		if err != nil {
			return errorResult("error consultando cuentas"), nil, nil
		}
		if len(accounts) == 0 {
			return textResult("El usuario no tiene ninguna cuenta registrada."), nil, nil
		}
		var sb strings.Builder
		for i, a := range accounts {
			if i > 0 {
				sb.WriteString("; ")
			}
			balance, err := tb.GetBalance(a.TBID)
			if err == nil {
				sb.WriteString(fmt.Sprintf("%s (%s): saldo %d", a.Number, a.Type, balance))
			} else {
				sb.WriteString(fmt.Sprintf("%s (%s)", a.Number, a.Type))
			}
		}
		return textResult(sb.String()), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolGetBalance,
		Description: "Consulta el saldo disponible de una cuenta bancaria del usuario autenticado.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, input accountNumberInput) (*gosdk.CallToolResult, any, error) {
		accounts, err := loadUserAccounts(ctx, pg, userID)
		if err != nil {
			return errorResult("error consultando cuentas"), nil, nil
		}
		acc, errResult := pickAccount(accounts, input.AccountNumber)
		if errResult != nil {
			return errResult, nil, nil
		}

		balance, err := tb.GetBalance(acc.TBID)
		if err != nil {
			return errorResult("error consultando saldo"), nil, nil
		}
		return textResult(fmt.Sprintf("El saldo de la cuenta %s (%s) es %d.", acc.Number, acc.Type, balance)), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolGetHistory,
		Description: "Consulta las ultimas transacciones de una cuenta bancaria del usuario autenticado.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, input accountNumberInput) (*gosdk.CallToolResult, any, error) {
		accounts, err := loadUserAccounts(ctx, pg, userID)
		if err != nil {
			return errorResult("error consultando cuentas"), nil, nil
		}
		acc, errResult := pickAccount(accounts, input.AccountNumber)
		if errResult != nil {
			return errResult, nil, nil
		}

		transfers, err := tb.GetHistory(acc.TBID, 5)
		if err != nil {
			return errorResult("error consultando historial"), nil, nil
		}
		if len(transfers) == 0 {
			return textResult(fmt.Sprintf("No hay transacciones registradas todavia en la cuenta %s.", acc.Number)), nil, nil
		}
		bankAccountID := tbtypes.ToUint128(db.BankAccountID)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Ultimos movimientos de la cuenta %s (%s): ", acc.Number, acc.Type))
		for i, t := range transfers {
			amtBig := t.Amount.BigInt()
			amt := amtBig.Int64()
			kind := "movimiento"
			switch {
			case t.DebitAccountID == bankAccountID:
				kind = "deposito"
			case t.CreditAccountID == bankAccountID:
				kind = "retiro"
			case t.DebitAccountID == acc.TBID:
				kind = "transferencia enviada"
			case t.CreditAccountID == acc.TBID:
				kind = "transferencia recibida"
			}
			if i > 0 {
				sb.WriteString("; ")
			}
			fecha := time.Unix(0, int64(t.Timestamp)).UTC().Format("2006-01-02 15:04 UTC")
			sb.WriteString(fmt.Sprintf("%s de %d el %s", kind, amt, fecha))
		}
		sb.WriteString(".")
		return textResult(sb.String()), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolDeposit,
		Description: "Deposita fondos en una cuenta bancaria del usuario autenticado. No es una operacion critica, se ejecuta directamente.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, input amountInput) (*gosdk.CallToolResult, any, error) {
		amount := uint64(input.Amount)
		if amount == 0 {
			return errorResult("el monto debe ser mayor a 0"), nil, nil
		}

		accounts, err := loadUserAccounts(ctx, pg, userID)
		if err != nil {
			return errorResult("error consultando cuentas"), nil, nil
		}
		acc, errResult := pickAccount(accounts, input.AccountNumber)
		if errResult != nil {
			return errResult, nil, nil
		}

		if err := tb.Deposit(acc.TBID, amount); err != nil {
			return errorResult("no se pudo procesar el deposito"), nil, nil
		}
		return textResult(fmt.Sprintf("Deposito de %d en la cuenta %s (%s) realizado correctamente.", amount, acc.Number, acc.Type)), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolWithdraw,
		Description: "Retira fondos de una cuenta bancaria del usuario autenticado. Es una operacion critica.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, input amountInput) (*gosdk.CallToolResult, any, error) {
		amount := uint64(input.Amount)
		if amount == 0 {
			return errorResult("el monto debe ser mayor a 0"), nil, nil
		}

		accounts, err := loadUserAccounts(ctx, pg, userID)
		if err != nil {
			return errorResult("error consultando cuentas"), nil, nil
		}
		acc, errResult := pickAccount(accounts, input.AccountNumber)
		if errResult != nil {
			return errResult, nil, nil
		}

		if err := tb.Withdraw(acc.TBID, amount); err != nil {
			if strings.Contains(err.Error(), "ExceedsCredits") {
				return errorResult("fondos insuficientes"), nil, nil
			}
			return errorResult("no se pudo procesar el retiro"), nil, nil
		}
		return textResult(fmt.Sprintf("Retiro de %d de la cuenta %s (%s) realizado correctamente.", amount, acc.Number, acc.Type)), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolTransfer,
		Description: "Transfiere fondos desde una cuenta del usuario autenticado hacia otra cuenta (propia o de un tercero). Es una operacion critica.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, input transferInput) (*gosdk.CallToolResult, any, error) {
		amount := uint64(input.Amount)
		if amount == 0 {
			return errorResult("el monto debe ser mayor a 0"), nil, nil
		}

		accounts, err := loadUserAccounts(ctx, pg, userID)
		if err != nil {
			return errorResult("error consultando cuentas"), nil, nil
		}
		fromAcc, errResult := pickAccount(accounts, input.FromAccountNumber)
		if errResult != nil {
			return errResult, nil, nil
		}

		var toTBHex string
		err = pg.QueryRow(ctx,
			`SELECT tigerbeetle_account_id FROM accounts WHERE account_number = $1`,
			input.ToAccountID,
		).Scan(&toTBHex)
		if err != nil {
			return errorResult("la cuenta destino no existe"), nil, nil
		}

		toID, err := tbtypes.HexStringToUint128(toTBHex)
		if err != nil {
			return errorResult("la cuenta destino no es valida"), nil, nil
		}
		if toID == fromAcc.TBID {
			return errorResult("no puedes transferir una cuenta a si misma"), nil, nil
		}
		if err := tb.Transfer(fromAcc.TBID, toID, amount); err != nil {
			if strings.Contains(err.Error(), "ExceedsCredits") {
				return errorResult("fondos insuficientes"), nil, nil
			}
			return errorResult("no se pudo procesar la transferencia"), nil, nil
		}
		return textResult(fmt.Sprintf("Transferencia de %d desde la cuenta %s (%s) hacia la cuenta %s realizada correctamente.", amount, fromAcc.Number, fromAcc.Type, input.ToAccountID)), nil, nil
	})

	return server
}
