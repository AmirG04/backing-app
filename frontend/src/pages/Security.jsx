import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'

export default function Security() {
  const [loading, setLoading] = useState(true)
  const [enabled, setEnabled] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')

  // Estado del flujo de activacion
  const [setupData, setSetupData] = useState(null) // { secret, otpauth_url }
  const [code, setCode] = useState('')

  // Estado del flujo de desactivacion
  const [showDisableForm, setShowDisableForm] = useState(false)
  const [password, setPassword] = useState('')

  function loadStatus() {
    setLoading(true)
    api
      .getAccountInfo()
      .then((data) => setEnabled(Boolean(data.two_factor_enabled)))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    loadStatus()
  }, [])

  async function startSetup() {
    setError('')
    setMessage('')
    try {
      const data = await api.setup2FA()
      setSetupData(data)
    } catch (err) {
      setError(err.message)
    }
  }

  async function confirmSetup(e) {
    e.preventDefault()
    setError('')
    try {
      await api.verify2FA(code)
      setSetupData(null)
      setCode('')
      setMessage('Autenticación en dos pasos activada correctamente.')
      loadStatus()
    } catch (err) {
      setError(err.message)
    }
  }

  async function handleDisable(e) {
    e.preventDefault()
    setError('')
    try {
      await api.disable2FA(password)
      setShowDisableForm(false)
      setPassword('')
      setMessage('Autenticación en dos pasos desactivada.')
      loadStatus()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div className="min-h-screen bg-slate-100">
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-2xl mx-auto px-4 py-4 flex items-center gap-4">
          <Link to="/dashboard" className="text-sm text-blue-600 font-medium">
            ← Volver
          </Link>
          <h1 className="text-lg font-bold text-slate-800">Seguridad</h1>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-4 py-8">
        <div className="bg-white rounded-xl shadow p-6">
          <h2 className="font-semibold text-slate-800 mb-1">
            Autenticación en dos pasos (2FA)
          </h2>
          <p className="text-sm text-slate-500 mb-4">
            Agrega una capa extra de seguridad: además de tu contraseña, se te
            pedirá un código de tu aplicación autenticadora (Google
            Authenticator, Authy, etc.) al iniciar sesión.
          </p>

          {message && (
            <div className="text-sm text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-lg px-3 py-2 mb-4">
              {message}
            </div>
          )}
          {error && (
            <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2 mb-4">
              {error}
            </div>
          )}

          {loading ? (
            <p className="text-sm text-slate-400">Cargando...</p>
          ) : setupData ? (
            /* --- Paso de confirmación: mostrar secreto + pedir primer código --- */
            <div className="space-y-4">
              <div>
                <p className="text-sm text-slate-600 mb-2">
                  Abre tu app autenticadora y agrega una cuenta nueva ingresando
                  esta clave manualmente:
                </p>
                <div className="bg-slate-100 rounded-lg px-4 py-3 font-mono text-center text-lg tracking-wider break-all">
                  {setupData.secret}
                </div>
                <a
                  href={setupData.otpauth_url}
                  className="text-xs text-blue-600 mt-2 inline-block"
                >
                  Abrir directamente en la app autenticadora (dispositivos móviles)
                </a>
              </div>

              <form onSubmit={confirmSetup} className="space-y-3">
                <label className="block text-sm font-medium text-slate-700">
                  Ingresa el código de 6 dígitos que te muestra la app para
                  confirmar:
                </label>
                <input
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]{6}"
                  maxLength={6}
                  placeholder="000000"
                  required
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
                  className="w-full text-center text-2xl tracking-[0.5em] rounded-lg border border-slate-300 px-3 py-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <div className="flex gap-2">
                  <button
                    type="submit"
                    disabled={code.length !== 6}
                    className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium py-2 rounded-lg transition"
                  >
                    Confirmar y activar
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setSetupData(null)
                      setCode('')
                    }}
                    className="px-4 text-sm text-slate-600 hover:text-slate-800"
                  >
                    Cancelar
                  </button>
                </div>
              </form>
            </div>
          ) : enabled ? (
            /* --- 2FA ya activo --- */
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-emerald-700 text-sm font-medium">
                <span className="w-2 h-2 rounded-full bg-emerald-500" />
                2FA está activo en tu cuenta
              </div>

              {!showDisableForm ? (
                <button
                  onClick={() => setShowDisableForm(true)}
                  className="text-sm text-red-600 hover:text-red-700 font-medium"
                >
                  Desactivar 2FA
                </button>
              ) : (
                <form onSubmit={handleDisable} className="space-y-3">
                  <label className="block text-sm font-medium text-slate-700">
                    Ingresa tu contraseña para confirmar:
                  </label>
                  <input
                    type="password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <div className="flex gap-2">
                    <button
                      type="submit"
                      className="flex-1 bg-red-600 hover:bg-red-700 text-white text-sm font-medium py-2 rounded-lg transition"
                    >
                      Desactivar
                    </button>
                    <button
                      type="button"
                      onClick={() => setShowDisableForm(false)}
                      className="px-4 text-sm text-slate-600 hover:text-slate-800"
                    >
                      Cancelar
                    </button>
                  </div>
                </form>
              )}
            </div>
          ) : (
            /* --- 2FA inactivo --- */
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-slate-500 text-sm">
                <span className="w-2 h-2 rounded-full bg-slate-300" />
                2FA no está activo
              </div>
              <button
                onClick={startSetup}
                className="bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition"
              >
                Activar 2FA
              </button>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
