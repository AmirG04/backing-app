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

	gosdk "github.com/modelcontextprotocol/go-sdk/mcp"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/db"
)

const (
	ToolGetBalance = "get_balance"
	ToolGetHistory = "get_history"
	ToolDeposit    = "deposit"
	ToolWithdraw   = "withdraw"
	ToolTransfer   = "transfer"
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

type amountInput struct {
	Amount float64 `json:"amount" jsonschema:"monto de la operacion, debe ser mayor a 0"`
}

type transferInput struct {
	ToAccountID string  `json:"to_account_id" jsonschema:"identificador hexadecimal de la cuenta destino"`
	Amount      float64 `json:"amount" jsonschema:"monto a transferir, debe ser mayor a 0"`
}

func textResult(msg string) *gosdk.CallToolResult {
	return &gosdk.CallToolResult{Content: []gosdk.Content{&gosdk.TextContent{Text: msg}}}
}

func errorResult(msg string) *gosdk.CallToolResult {
	return &gosdk.CallToolResult{IsError: true, Content: []gosdk.Content{&gosdk.TextContent{Text: msg}}}
}

// NewBankingServer crea un servidor MCP con las herramientas bancarias
// atadas a la cuenta de TigerBeetle de un usuario especifico. Se crea uno
// nuevo por cada conversacion de chat (son objetos livianos), de forma
// que las herramientas solo puedan operar sobre la cuenta del usuario
// autenticado en esa peticion - nunca sobre la de otro usuario.
func NewBankingServer(tb *db.TigerBeetleClient, accountID tbtypes.Uint128) *gosdk.Server {
	server := gosdk.NewServer(&gosdk.Implementation{Name: "banking-mcp-server", Version: "v1.0.0"}, nil)

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolGetBalance,
		Description: "Consulta el saldo disponible en la cuenta bancaria del usuario autenticado. No requiere argumentos.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, _ emptyInput) (*gosdk.CallToolResult, any, error) {
		balance, err := tb.GetBalance(accountID)
		if err != nil {
			return errorResult("error consultando saldo"), nil, nil
		}
		return textResult(fmt.Sprintf("El saldo disponible es %d.", balance)), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolGetHistory,
		Description: "Consulta las ultimas transacciones (depositos, retiros y transferencias) del usuario autenticado. No requiere argumentos.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, _ emptyInput) (*gosdk.CallToolResult, any, error) {
		transfers, err := tb.GetHistory(accountID, 5)
		if err != nil {
			return errorResult("error consultando historial"), nil, nil
		}
		if len(transfers) == 0 {
			return textResult("No hay transacciones registradas todavia."), nil, nil
		}
		bankAccountID := tbtypes.ToUint128(db.BankAccountID)
		var sb strings.Builder
		sb.WriteString("Ultimos movimientos: ")
		for i, t := range transfers {
			amtBig := t.Amount.BigInt()
			amt := amtBig.Int64()
			kind := "movimiento"
			switch {
			case t.DebitAccountID == bankAccountID:
				kind = "deposito"
			case t.CreditAccountID == bankAccountID:
				kind = "retiro"
			case t.DebitAccountID == accountID:
				kind = "transferencia enviada"
			case t.CreditAccountID == accountID:
				kind = "transferencia recibida"
			}
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(fmt.Sprintf("%s de %d", kind, amt))
		}
		sb.WriteString(".")
		return textResult(sb.String()), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolDeposit,
		Description: "Deposita fondos en la cuenta del usuario autenticado. No es una operacion critica, se ejecuta directamente.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, input amountInput) (*gosdk.CallToolResult, any, error) {
		amount := uint64(input.Amount)
		if amount == 0 {
			return errorResult("el monto debe ser mayor a 0"), nil, nil
		}
		if err := tb.Deposit(accountID, amount); err != nil {
			return errorResult("no se pudo procesar el deposito"), nil, nil
		}
		return textResult(fmt.Sprintf("Deposito de %d realizado correctamente.", amount)), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolWithdraw,
		Description: "Retira fondos de la cuenta del usuario autenticado. Es una operacion critica.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, input amountInput) (*gosdk.CallToolResult, any, error) {
		amount := uint64(input.Amount)
		if amount == 0 {
			return errorResult("el monto debe ser mayor a 0"), nil, nil
		}
		if err := tb.Withdraw(accountID, amount); err != nil {
			if strings.Contains(err.Error(), "ExceedsCredits") {
				return errorResult("fondos insuficientes"), nil, nil
			}
			return errorResult("no se pudo procesar el retiro"), nil, nil
		}
		return textResult(fmt.Sprintf("Retiro de %d realizado correctamente.", amount)), nil, nil
	})

	gosdk.AddTool(server, &gosdk.Tool{
		Name:        ToolTransfer,
		Description: "Transfiere fondos de la cuenta del usuario autenticado a otra cuenta. Es una operacion critica.",
	}, func(ctx context.Context, req *gosdk.CallToolRequest, input transferInput) (*gosdk.CallToolResult, any, error) {
		amount := uint64(input.Amount)
		if amount == 0 {
			return errorResult("el monto debe ser mayor a 0"), nil, nil
		}
		toID, err := tbtypes.HexStringToUint128(input.ToAccountID)
		if err != nil {
			return errorResult("la cuenta destino no es valida"), nil, nil
		}
		if toID == accountID {
			return errorResult("no puedes transferir a tu propia cuenta"), nil, nil
		}
		if err := tb.Transfer(accountID, toID, amount); err != nil {
			if strings.Contains(err.Error(), "ExceedsCredits") {
				return errorResult("fondos insuficientes"), nil, nil
			}
			return errorResult("no se pudo procesar la transferencia"), nil, nil
		}
		return textResult(fmt.Sprintf("Transferencia de %d a la cuenta %s realizada correctamente.", amount, input.ToAccountID)), nil, nil
	})

	return server
}
