package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gosdk "github.com/modelcontextprotocol/go-sdk/mcp"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	"banking-app/backend/internal/ai"
	"banking-app/backend/internal/mcp"
	"banking-app/backend/internal/middleware"
	"banking-app/backend/internal/models"
)

const chatSystemPrompt = `Eres un asistente bancario para una aplicacion de banca simplificada.
Ayudas al usuario autenticado a consultar su saldo, revisar su historial de
transacciones, depositar, retirar y transferir dinero, usando exclusivamente
las herramientas disponibles (nunca inventes montos ni saldos).

Responde siempre en español, de forma breve, clara y amigable.

REGLAS CRITICAS - sigue estas reglas sin excepcion:
1. NUNCA afirmes que un deposito, retiro o transferencia se completo a
   menos que hayas recibido un resultado de herramienta que confirme
   exactamente esa operacion en este turno. Si no invocaste la
   herramienta o no recibiste su resultado, no puedes decir que la
   operacion se realizo.
2. Invoca como maximo UNA herramienta por mensaje del usuario. Si el
   usuario pide varias cosas en el mismo mensaje (por ejemplo "cuanto
   tengo y retira 10"), procesa solo la primera y termina tu respuesta
   pidiendole que solicite la segunda en un mensaje separado - no
   invoques una segunda herramienta en el mismo turno.
3. Antes de invocar "withdraw" o "transfer" necesitas tener claro el
   monto (y la cuenta destino en el caso de transferencias). Si el
   usuario no dio esa informacion todavia, pidesela en un mensaje de
   texto en vez de invocar la herramienta.
4. El sistema se encarga automaticamente de pedirle confirmacion final
   al usuario antes de ejecutar cualquier operacion critica (withdraw o
   transfer), asi que no necesitas pedir confirmacion tu mismo - solo
   invoca la herramienta cuando tengas los datos necesarios.

FORMATO Y TONO:
- Usa un tono calido y cercano, como un asistente que realmente quiere
  ayudar - no un sistema robotico.
- Da formato a tus respuestas con markdown cuando ayude a la claridad:
  negritas para montos importantes, listas numeradas o con vinetas para
  historiales o pasos, emojis ocasionales y relevantes (💰 📋 ✅ etc.)
  sin exagerar.
- Se breve: nadie quiere leer un parrafo largo para saber su saldo.
- Cuando muestres montos, usa el simbolo $ (ej. "$100").
- Termina con una pregunta o pie amigable solo cuando tenga sentido
  (no lo agregues mecanicamente en cada respuesta).`

// POST /api/chat
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "el campo 'message' es requerido")
		return
	}

	accountID, _, err := h.getUserTBAccountID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "cuenta no encontrada")
		return
	}

	ctx := r.Context()

	// Conectamos un cliente MCP a un servidor MCP nuevo, atado a la cuenta
	// de este usuario. Server y client viven en el mismo proceso, pero se
	// comunican por el protocolo MCP real (JSON-RPC) a traves de un
	// transporte en memoria que provee el SDK - no son llamadas directas
	// a funciones Go.
	mcpSession, closeSession, err := h.connectMCP(ctx, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error inicializando MCP: "+err.Error())
		return
	}
	defer closeSession()

	// 1. Si hay una operacion critica esperando confirmacion, resolverla
	// antes de mandar nada al modelo.
	h.pendingMu.Lock()
	pending, hasPending := h.pendingByUser[userID]
	h.pendingMu.Unlock()

	if hasPending {
		switch {
		case isConfirmation(req.Message):
			result, execErr := callMCPTool(ctx, mcpSession, pending.ToolName, pending.Args)
			h.clearPending(userID)
			if execErr != nil {
				writeJSON(w, http.StatusOK, models.ChatResponse{
					Reply: fmt.Sprintf("No se pudo completar la operacion: %s", execErr.Error()),
				})
				return
			}
			writeJSON(w, http.StatusOK, models.ChatResponse{Reply: result, ActionExecuted: true})
			return

		case isCancellation(req.Message):
			h.clearPending(userID)
			writeJSON(w, http.StatusOK, models.ChatResponse{Reply: "Operacion cancelada, no se realizo ningun cambio."})
			return

		default:
			h.clearPending(userID)
		}
	}

	// 2. Descubrir las herramientas disponibles hablando el protocolo MCP
	// (tools/list) en vez de tener un catalogo hardcodeado del lado del chat.
	tools, err := listMCPTools(ctx, mcpSession)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error listando herramientas MCP: "+err.Error())
		return
	}

	messages := []ai.Message{
		{Role: "system", Content: chatSystemPrompt},
		{Role: "user", Content: req.Message},
	}

	resp, err := h.AI.SendMessage(messages, tools)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error consultando el modelo de IA: "+err.Error())
		return
	}

	toolCall, textReply := extractToolCall(resp)
	if toolCall == nil {
		writeJSON(w, http.StatusOK, models.ChatResponse{Reply: textReply})
		return
	}

	var argsMap map[string]any
	_ = json.Unmarshal([]byte(toolCall.Function.Arguments), &argsMap)

	// 3. Operaciones criticas: pedir confirmacion en vez de ejecutar.
	if mcp.CriticalTools[toolCall.Function.Name] {
		h.pendingMu.Lock()
		h.pendingByUser[userID] = pendingAction{
			ToolName: toolCall.Function.Name,
			Args:     argsMap,
		}
		h.pendingMu.Unlock()

		writeJSON(w, http.StatusOK, models.ChatResponse{
			Reply:                buildConfirmationMessage(toolCall.Function.Name, argsMap),
			RequiresConfirmation: true,
		})
		return
	}

	// 4. Operaciones no criticas (saldo, historial, deposito): ejecutar
	// llamando a la herramienta MCP real (tools/call).
	result, execErr := callMCPTool(ctx, mcpSession, toolCall.Function.Name, argsMap)
	toolResultText := result
	if execErr != nil {
		toolResultText = execErr.Error()
	}

	// 5. Mandar el resultado de vuelta al modelo para que redacte la
	// respuesta final en lenguaje natural.
	assistantMsg := resp.Choices[0].Message

	toolResultMessages := []ai.Message{
		{Role: "tool", ToolCallID: toolCall.ID, Content: toolResultText},
	}
	// Defensa contra alucinaciones: si el modelo pidio mas de una
	// herramienta en el mismo turno (ej. "cuanto tengo y retira 10"),
	// solo procesamos la primera. Para las demas, respondemos
	// explicitamente que no se ejecutaron, en vez de dejarlas sin
	// respuesta - de lo contrario el modelo puede inventarse un
	// resultado plausible para ellas en su respuesta final.
	for _, extraCall := range assistantMsg.ToolCalls[1:] {
		toolResultMessages = append(toolResultMessages, ai.Message{
			Role:       "tool",
			ToolCallID: extraCall.ID,
			Content:    "Esta operacion no se proceso. Solo se puede procesar una operacion por mensaje; pide al usuario que la solicite por separado.",
		})
	}

	followUpMessages := append(messages, assistantMsg)
	followUpMessages = append(followUpMessages, toolResultMessages...)

	followUp, err := h.AI.SendMessage(followUpMessages, tools)
	if err != nil {
		writeJSON(w, http.StatusOK, models.ChatResponse{Reply: toolResultText, ActionExecuted: execErr == nil})
		return
	}

	_, finalText := extractToolCall(followUp)
	writeJSON(w, http.StatusOK, models.ChatResponse{Reply: finalText, ActionExecuted: execErr == nil})
}

// connectMCP levanta un servidor MCP para la cuenta del usuario, conecta
// un cliente a el sobre un transporte en memoria, y retorna la sesion del
// cliente lista para usarse, junto con una funcion de limpieza.
func (h *Handler) connectMCP(ctx context.Context, accountID tbtypes.Uint128) (*gosdk.ClientSession, func(), error) {
	server := mcp.NewBankingServer(h.TB, accountID)
	client := gosdk.NewClient(&gosdk.Implementation{Name: "banking-chat-client", Version: "v1.0.0"}, nil)

	serverTransport, clientTransport := gosdk.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, func() {}, fmt.Errorf("error conectando servidor mcp: %w", err)
	}

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, func() {}, fmt.Errorf("error conectando cliente mcp: %w", err)
	}

	cleanup := func() {
		clientSession.Close()
		serverSession.Close()
	}
	return clientSession, cleanup, nil
}

// listMCPTools pide al servidor MCP su catalogo de herramientas
// (tools/list) y lo convierte al formato "function calling" de OpenRouter.
func listMCPTools(ctx context.Context, session *gosdk.ClientSession) ([]ai.Tool, error) {
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	tools := make([]ai.Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		schemaBytes, err := json.Marshal(t.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("error serializando schema de %s: %w", t.Name, err)
		}
		tools = append(tools, ai.Tool{
			Type: "function",
			Function: ai.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schemaBytes,
			},
		})
	}
	return tools, nil
}

// callMCPTool invoca una herramienta en el servidor MCP (tools/call) y
// devuelve el texto de la respuesta.
func callMCPTool(ctx context.Context, session *gosdk.ClientSession, toolName string, args map[string]any) (string, error) {
	result, err := session.CallTool(ctx, &gosdk.CallToolParams{Name: toolName, Arguments: args})
	if err != nil {
		return "", fmt.Errorf("error ejecutando la herramienta")
	}

	var text string
	for _, c := range result.Content {
		if tc, ok := c.(*gosdk.TextContent); ok {
			text = tc.Text
			break
		}
	}
	if text == "" {
		text = "Operacion completada."
	}
	if result.IsError {
		return "", fmt.Errorf("%s", text)
	}
	return text, nil
}

// extractToolCall revisa la primera respuesta del modelo: si decidio
// invocar una herramienta, la retorna; si no, retorna su texto.
func extractToolCall(resp *ai.Response) (*ai.ToolCall, string) {
	if len(resp.Choices) == 0 {
		return nil, "Lo siento, no pude procesar tu mensaje."
	}
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) > 0 {
		return &msg.ToolCalls[0], msg.Content
	}
	reply := msg.Content
	if reply == "" {
		reply = "Listo."
	}
	return nil, reply
}

func (h *Handler) clearPending(userID string) {
	h.pendingMu.Lock()
	delete(h.pendingByUser, userID)
	h.pendingMu.Unlock()
}

func buildConfirmationMessage(toolName string, args map[string]any) string {
	amount := 0.0
	if a, ok := args["amount"].(float64); ok {
		amount = a
	}
	switch toolName {
	case mcp.ToolWithdraw:
		return fmt.Sprintf("¿Confirmas que deseas retirar %.0f? Responde \"si\" para confirmar o \"no\" para cancelar.", amount)
	case mcp.ToolTransfer:
		toAccount, _ := args["to_account_id"].(string)
		return fmt.Sprintf("¿Confirmas que deseas transferir %.0f a la cuenta %s? Responde \"si\" para confirmar o \"no\" para cancelar.", amount, toAccount)
	default:
		return "¿Confirmas esta operacion? Responde \"si\" para confirmar o \"no\" para cancelar."
	}
}

func isConfirmation(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	for _, k := range []string{"si", "sí", "confirmo", "confirmar", "yes", "dale", "adelante", "correcto", "hazlo"} {
		if m == k || strings.Contains(m, k) {
			return true
		}
	}
	return false
}

func isCancellation(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	for _, k := range []string{"no", "cancela", "cancelar", "detente", "olvidalo"} {
		if m == k || strings.Contains(m, k) {
			return true
		}
	}
	return false
}
