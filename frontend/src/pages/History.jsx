import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'

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
                      {t.debit_account_id === t.credit_account_id
                        ? 'Movimiento'
                        : 'Transferencia'}
                    </p>
                    <p className="text-xs text-slate-400">{t.timestamp}</p>
                  </div>
                  <span className="font-semibold text-slate-800">{t.amount}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </main>
    </div>
  )
}
