package handlers

import "net/http"

// POST /api/chat
// TODO: implementar en la fase de integracion MCP + modelo de IA.
// Aqui se recibira el mensaje del usuario, se le pasara al modelo junto
// con las tools disponibles (deposit, withdraw, transfer, get_balance,
// get_history), y se ejecutara la tool que el modelo decida invocar -
// pidiendo confirmacion explicita para acciones criticas (retiro/transferencia).
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "chat con IA: pendiente de implementar")
}
