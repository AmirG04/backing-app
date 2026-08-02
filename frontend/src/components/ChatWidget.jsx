import React, { useState, useRef, useEffect } from 'react'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// Renderer de markdown minimo (sin dependencias externas): soporta
// **negritas** y lineas de lista (numeradas o con "-"), que es lo unico
// que el system prompt del asistente usa. Cualquier otro texto se
// muestra tal cual, con saltos de linea respetados.
function renderMarkdownLine(line, key) {
  const parts = line.split(/(\*\*[^*]+\*\*)/g).filter(Boolean)
  return (
    <React.Fragment key={key}>
      {parts.map((part, i) =>
        part.startsWith('**') && part.endsWith('**') ? (
          <strong key={i}>{part.slice(2, -2)}</strong>
        ) : (
          <React.Fragment key={i}>{part}</React.Fragment>
        )
      )}
    </React.Fragment>
  )
}

function ChatMessageContent({ text }) {
  const lines = text.split('\n')
  return (
    <div className="space-y-0.5">
      {lines.map((line, i) => {
        const trimmed = line.trim()
        const isListItem = /^(\d+\.|-)\s+/.test(trimmed)
        return (
          <div key={i} className={isListItem ? 'pl-1' : ''}>
            {renderMarkdownLine(line, i)}
          </div>
        )
      })}
    </div>
  )
}

export default function ChatWidget({ onActionCompleted }) {
  const [messages, setMessages] = useState([
    {
      role: 'assistant',
      content:
        'Hola, soy tu asistente bancario. Puedes pedirme cosas como "¿cuánto dinero tengo?" o "transfiere $50 a la cuenta 123".',
    },
  ])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const bottomRef = useRef(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  async function sendMessage(e) {
    e.preventDefault()
    if (!input.trim()) return

    const userMessage = { role: 'user', content: input }
    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setLoading(true)

    try {
      const token = localStorage.getItem('token')
      const res = await fetch(`${API_URL}/api/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ message: userMessage.content }),
      })
      const data = await res.json()

      if (!res.ok) {
        throw new Error(data.error || 'El chat aún no está disponible')
      }

      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: data.reply, needsConfirmation: data.requires_confirmation },
      ])
      if (data.action_executed) {
        onActionCompleted?.()
      }
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: `⚠️ ${err.message}` },
      ])
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="bg-white rounded-xl shadow flex flex-col h-[520px]">
      <div className="px-4 py-3 border-b border-slate-100">
        <h2 className="font-semibold text-slate-800">Asistente bancario</h2>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
        {messages.map((m, i) => (
          <div
            key={i}
            className={`text-sm max-w-[85%] px-3 py-2 rounded-lg ${
              m.role === 'user'
                ? 'bg-blue-600 text-white ml-auto'
                : m.needsConfirmation
                ? 'bg-amber-50 text-amber-800 border border-amber-200'
                : 'bg-slate-100 text-slate-700'
            }`}
          >
            <ChatMessageContent text={m.content} />
          </div>
        ))}
        {loading && <div className="text-xs text-slate-400">Escribiendo...</div>}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={sendMessage} className="p-3 border-t border-slate-100 flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Escribe un mensaje..."
          className="flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          type="submit"
          disabled={loading}
          className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium px-4 rounded-lg transition"
        >
          Enviar
        </button>
      </form>
    </div>
  )
}
