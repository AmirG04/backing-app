import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'

const TYPE_LABELS = {
  deposit: 'Depósito',
  withdrawal: 'Retiro',
  transfer_sent: 'Transferencia enviada',
  transfer_received: 'Transferencia recibida',
  unknown: 'Movimiento',
}

export default function History() {
  const [transactions, setTransactions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .history()
      .then((data) => setTransactions(Array.isArray(data) ? data : []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="min-h-screen bg-slate-100">
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-3xl mx-auto px-4 py-4 flex items-center gap-4">
          <Link to="/dashboard" className="text-sm text-blue-600 font-medium">
            ← Volver
          </Link>
          <h1 className="text-lg font-bold text-slate-800">Historial de transacciones</h1>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-4 py-8">
        <div className="bg-white rounded-xl shadow p-6">
          {loading ? (
            <p className="text-sm text-slate-400">Cargando...</p>
          ) : error ? (
            <p className="text-sm text-red-600">{error}</p>
          ) : transactions.length === 0 ? (
            <p className="text-sm text-slate-400">No hay transacciones todavía.</p>
          ) : (
            <ul className="divide-y divide-slate-100">
              {transactions.map((t) => (
                <li key={t.id} className="py-3 flex justify-between text-sm">
                  <div>
                    <p className="font-medium text-slate-800">
                      {TYPE_LABELS[t.type] || t.type}
                    </p>
                    <p className="text-xs text-slate-400">
                      {new Date(t.timestamp).toLocaleString()}
                    </p>
                  </div>
                  <span
                    className={`font-semibold ${
                      t.type === 'deposit' || t.type === 'transfer_received'
                        ? 'text-emerald-600'
                        : 'text-slate-800'
                    }`}
                  >
                    {t.type === 'deposit' || t.type === 'transfer_received' ? '+' : '-'}
                    {t.amount}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </main>
    </div>
  )
}
