import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../lib/auth'
import { api } from '../lib/api'
import ChatWidget from '../components/ChatWidget'

export default function Dashboard() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const [balance, setBalance] = useState(null)
  const [recent, setRecent] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [amount, setAmount] = useState('')
  const [toAccount, setToAccount] = useState('')
  const [actionLoading, setActionLoading] = useState(false)
  const [message, setMessage] = useState('')

  async function loadData() {
    setLoading(true)
    setError('')
    try {
      const [balanceData, historyData] = await Promise.all([api.getBalance(), api.history()])
      setBalance(balanceData)
      setRecent(Array.isArray(historyData) ? historyData.slice(0, 5) : [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  function handleLogout() {
    logout()
    navigate('/login')
  }

  async function runAction(fn) {
    setMessage('')
    setError('')
    setActionLoading(true)
    try {
      await fn()
      setMessage('Operación realizada con éxito')
      setAmount('')
      setToAccount('')
      loadData()
    } catch (err) {
      setError(err.message)
    } finally {
      setActionLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-100">
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-5xl mx-auto px-4 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-lg font-bold text-slate-800">Banco Simplificado</h1>
            <p className="text-sm text-slate-500">Hola, {user?.full_name}</p>
          </div>
          <button
            onClick={handleLogout}
            className="text-sm text-slate-600 hover:text-red-600 font-medium"
          >
            Cerrar sesión
          </button>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-4 py-8 grid gap-6 md:grid-cols-3">
        {/* Columna principal */}
        <div className="md:col-span-2 space-y-6">
          {/* Tarjeta de saldo */}
          <div className="bg-white rounded-xl shadow p-6">
            <p className="text-sm text-slate-500">Saldo disponible</p>
            {loading ? (
              <div className="h-9 w-40 bg-slate-200 rounded animate-pulse mt-2" />
            ) : (
              <p className="text-3xl font-bold text-slate-800 mt-1">
                ${balance ? balance.balance.toLocaleString() : '0'}
              </p>
            )}
            <p className="text-xs text-slate-400 mt-2 break-all">
              Cuenta: {balance?.account_id || user?.tigerbeetle_account_id}
            </p>
          </div>

          {/* Operaciones */}
          <div className="bg-white rounded-xl shadow p-6">
            <h2 className="font-semibold text-slate-800 mb-4">Realizar operación</h2>

            <div className="space-y-3">
              <input
                type="number"
                placeholder="Monto"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <input
                type="text"
                placeholder="ID de cuenta destino (solo para transferencia)"
                value={toAccount}
                onChange={(e) => setToAccount(e.target.value)}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />

              <div className="flex flex-wrap gap-2">
                <button
                  disabled={actionLoading || !amount}
                  onClick={() => runAction(() => api.deposit(Number(amount)))}
                  className="flex-1 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white text-sm font-medium py-2 rounded-lg transition"
                >
                  Depositar
                </button>
                <button
                  disabled={actionLoading || !amount}
                  onClick={() => runAction(() => api.withdraw(Number(amount)))}
                  className="flex-1 bg-amber-600 hover:bg-amber-700 disabled:opacity-50 text-white text-sm font-medium py-2 rounded-lg transition"
                >
                  Retirar
                </button>
                <button
                  disabled={actionLoading || !amount || !toAccount}
                  onClick={() => runAction(() => api.transfer(toAccount, Number(amount)))}
                  className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium py-2 rounded-lg transition"
                >
                  Transferir
                </button>
              </div>

              {message && (
                <div className="text-sm text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-lg px-3 py-2">
                  {message}
                </div>
              )}
              {error && (
                <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
                  {error}
                </div>
              )}
            </div>
          </div>

          {/* Transacciones recientes */}
          <div className="bg-white rounded-xl shadow p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-slate-800">Transacciones recientes</h2>
              <a href="/history" className="text-sm text-blue-600 font-medium">
                Ver todo
              </a>
            </div>

            {loading ? (
              <p className="text-sm text-slate-400">Cargando...</p>
            ) : recent.length === 0 ? (
              <p className="text-sm text-slate-400">Aún no tienes transacciones.</p>
            ) : (
              <ul className="divide-y divide-slate-100">
                {recent.map((t) => (
                  <li key={t.id} className="py-2 text-sm flex justify-between">
                    <span className="text-slate-600">{t.code === 1 ? 'Movimiento' : 'Otro'}</span>
                    <span className="font-medium text-slate-800">{t.amount}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        {/* Chat con IA */}
        <div className="md:col-span-1">
          <ChatWidget onActionCompleted={loadData} />
        </div>
      </main>
    </div>
  )
}
