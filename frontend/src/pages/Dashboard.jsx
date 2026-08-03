import React, { useEffect, useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../lib/auth'
import { api } from '../lib/api'
import ChatWidget from '../components/ChatWidget'

const ACCOUNT_TYPE_LABELS = {
  checking: 'Corriente',
  savings: 'Ahorro',
}

export default function Dashboard() {
  const { user, account, logout } = useAuth()
  const navigate = useNavigate()

  const [accounts, setAccounts] = useState([])
  const [selectedAccountId, setSelectedAccountId] = useState(null)

  const [balance, setBalance] = useState(null)
  const [recent, setRecent] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [amount, setAmount] = useState('')
  const [toAccount, setToAccount] = useState('')
  const [actionLoading, setActionLoading] = useState(false)
  const [message, setMessage] = useState('')

  // El mensaje de exito desaparece solo despues de unos segundos, en vez
  // de quedarse pegado hasta la siguiente accion.
  useEffect(() => {
    if (!message) return
    const timer = setTimeout(() => setMessage(''), 4000)
    return () => clearTimeout(timer)
  }, [message])

  const [showCreateAccount, setShowCreateAccount] = useState(false)
  const [newAccountType, setNewAccountType] = useState('checking')
  const [creatingAccount, setCreatingAccount] = useState(false)

  // Carga la lista de cuentas del usuario. keepSelection=true evita
  // cambiar la cuenta activa si ya habia una elegida (ej. al refrescar
  // despues de crear una cuenta nueva).
  async function refreshAccounts(keepSelection) {
    try {
      const data = await api.getAccounts()
      const list = Array.isArray(data) ? data : []
      setAccounts(list)
      if (!keepSelection || !selectedAccountId) {
        const defaultId = account?.id || list[0]?.id || null
        setSelectedAccountId(defaultId)
      }
      return list
    } catch {
      setSelectedAccountId((prev) => prev || account?.id || null)
      return []
    }
  }

  useEffect(() => {
    refreshAccounts(false)
  }, [])

  async function loadData(accountId) {
    setLoading(true)
    setError('')
    try {
      const [balanceData, historyData] = await Promise.all([
        api.getBalance(accountId),
        api.history(accountId),
      ])
      setBalance(balanceData)
      setRecent(Array.isArray(historyData) ? historyData.slice(0, 5) : [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (selectedAccountId) {
      loadData(selectedAccountId)
    }
  }, [selectedAccountId])

  function handleLogout() {
    logout()
    navigate('/login')
  }

  async function handleCreateAccount(e) {
    e.preventDefault()
    setError('')
    setMessage('')
    setCreatingAccount(true)
    try {
      const created = await api.createAccount(newAccountType)
      await refreshAccounts(true)
      setSelectedAccountId(created.id)
      setShowCreateAccount(false)
      setMessage(`Cuenta ${ACCOUNT_TYPE_LABELS[newAccountType] || newAccountType} creada: ${created.account_number}`)
    } catch (err) {
      setError(err.message)
    } finally {
      setCreatingAccount(false)
    }
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
      loadData(selectedAccountId)
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
          <div className="flex items-center gap-4">
            <Link to="/security" className="text-sm text-slate-600 hover:text-blue-600 font-medium">
              Seguridad
            </Link>
            <button
              onClick={handleLogout}
              className="text-sm text-slate-600 hover:text-red-600 font-medium"
            >
              Cerrar sesión
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-4 py-8 grid gap-6 md:grid-cols-3">
        {/* Columna principal */}
        <div className="md:col-span-2 space-y-6">
          {/* Tarjeta de saldo + selector de cuentas */}
          <div className="bg-white rounded-xl shadow p-6">
            <div className="flex items-start justify-between gap-4 flex-wrap">
              <div>
                <p className="text-sm text-slate-500">Saldo disponible</p>
                {loading ? (
                  <div className="h-9 w-40 bg-slate-200 rounded animate-pulse mt-2" />
                ) : (
                  <p className="text-3xl font-bold text-slate-800 mt-1">
                    ${balance ? balance.balance.toLocaleString() : '0'}
                  </p>
                )}
                <p className="text-xs text-slate-400 mt-2 break-all">
                  Cuenta: {balance?.account_number}
                </p>
              </div>

              {accounts.length > 1 && (
                <div className="min-w-[180px]">
                  <label className="block text-xs text-slate-500 mb-1">Cuenta activa</label>
                  <select
                    value={selectedAccountId || ''}
                    onChange={(e) => setSelectedAccountId(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    {accounts.map((acc) => (
                      <option key={acc.id} value={acc.id}>
                        {ACCOUNT_TYPE_LABELS[acc.account_type] || acc.account_type}
                        {acc.account_number ? ` · ${acc.account_number}` : ''}
                      </option>
                    ))}
                  </select>
                </div>
              )}
            </div>

            {!showCreateAccount ? (
              <button
                onClick={() => setShowCreateAccount(true)}
                className="text-sm text-blue-600 hover:text-blue-700 font-medium mt-4"
              >
                + Abrir una cuenta nueva
              </button>
            ) : (
              <form onSubmit={handleCreateAccount} className="mt-4 flex flex-wrap items-end gap-2">
                <div>
                  <label className="block text-xs text-slate-500 mb-1">Tipo de cuenta</label>
                  <select
                    value={newAccountType}
                    onChange={(e) => setNewAccountType(e.target.value)}
                    className="rounded-lg border border-slate-300 px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="checking">Corriente</option>
                    <option value="savings">Ahorro</option>
                  </select>
                </div>
                <button
                  type="submit"
                  disabled={creatingAccount}
                  className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium px-4 py-1.5 rounded-lg transition"
                >
                  {creatingAccount ? 'Creando...' : 'Crear cuenta'}
                </button>
                <button
                  type="button"
                  onClick={() => setShowCreateAccount(false)}
                  className="text-sm text-slate-500 hover:text-slate-700 px-2 py-1.5"
                >
                  Cancelar
                </button>
              </form>
            )}
          </div>

          {/* Operaciones */}
          <div className="bg-white rounded-xl shadow p-6">
            <h2 className="font-semibold text-slate-800 mb-4">Realizar operación</h2>

            <div className="space-y-3">
              <input
                type="number"
                min="1"
                step="1"
                placeholder="Monto"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <input
                type="text"
                placeholder="Número de cuenta destino (ej. 4001-1234-5678-0001)"
                value={toAccount}
                onChange={(e) => setToAccount(e.target.value)}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />

              <div className="flex flex-wrap gap-2">
                <button
                  disabled={actionLoading || !amount || Number(amount) <= 0}
                  onClick={() => runAction(() => api.deposit(Number(amount), selectedAccountId))}
                  className="flex-1 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white text-sm font-medium py-2 rounded-lg transition"
                >
                  Depositar
                </button>
                <button
                  disabled={actionLoading || !amount || Number(amount) <= 0}
                  onClick={() => runAction(() => api.withdraw(Number(amount), selectedAccountId))}
                  className="flex-1 bg-amber-600 hover:bg-amber-700 disabled:opacity-50 text-white text-sm font-medium py-2 rounded-lg transition"
                >
                  Retirar
                </button>
                <button
                  disabled={actionLoading || !amount || Number(amount) <= 0 || !toAccount}
                  onClick={() =>
                    runAction(() => api.transfer(toAccount, Number(amount), selectedAccountId))
                  }
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
                    <span className="text-slate-600">
                      {t.type === 'deposit' && 'Depósito'}
                      {t.type === 'withdrawal' && 'Retiro'}
                      {t.type === 'transfer_sent' && 'Transferencia enviada'}
                      {t.type === 'transfer_received' && 'Transferencia recibida'}
                    </span>
                    <span
                      className={`font-medium ${
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
        </div>

        {/* Chat con IA */}
        <div className="md:col-span-1">
          <ChatWidget onActionCompleted={() => loadData(selectedAccountId)} />
        </div>
      </main>
    </div>
  )
}
