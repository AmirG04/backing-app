import React, { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../lib/auth'

export default function Login({ mode = 'login' }) {
  const isRegister = mode === 'register'
  const { login, completeTwoFactorLogin, register } = useAuth()
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [fullName, setFullName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [infoMessage, setInfoMessage] = useState('')

  useEffect(() => {
    if (sessionStorage.getItem('logout_reason') === 'inactivity') {
      setInfoMessage('Tu sesión se cerró automáticamente por inactividad. Inicia sesión de nuevo.')
      sessionStorage.removeItem('logout_reason')
    }
  }, [])

  // Si el login requiere 2FA, guardamos el token temporal aqui y
  // mostramos un segundo formulario pidiendo el codigo de 6 digitos.
  const [pendingPreAuthToken, setPendingPreAuthToken] = useState(null)
  const [code, setCode] = useState('')

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      if (isRegister) {
        await register(email, password, fullName)
        navigate('/dashboard')
        return
      }
      const result = await login(email, password)
      if (result.requiresTwoFactor) {
        setPendingPreAuthToken(result.preAuthToken)
      } else {
        navigate('/dashboard')
      }
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function handleTwoFactorSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await completeTwoFactorLogin(pendingPreAuthToken, code)
      navigate('/dashboard')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  // --- Paso 2: pedir el codigo de 6 digitos (solo si el login lo requirio) ---
  if (pendingPreAuthToken) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-100 px-4">
        <div className="w-full max-w-sm bg-white rounded-xl shadow p-8">
          <h1 className="text-2xl font-bold text-slate-800 mb-1">Verificación en dos pasos</h1>
          <p className="text-slate-500 text-sm mb-6">
            Ingresa el código de 6 dígitos de tu aplicación autenticadora.
          </p>

          <form onSubmit={handleTwoFactorSubmit} className="space-y-4">
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]{6}"
              maxLength={6}
              placeholder="000000"
              required
              autoFocus
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
              className="w-full text-center text-2xl tracking-[0.5em] rounded-lg border border-slate-300 px-3 py-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />

            {error && (
              <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading || code.length !== 6}
              className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white font-medium py-2 rounded-lg transition"
            >
              {loading ? 'Verificando...' : 'Verificar'}
            </button>
          </form>

          <button
            onClick={() => {
              setPendingPreAuthToken(null)
              setCode('')
              setError('')
            }}
            className="text-sm text-slate-500 mt-4 w-full text-center hover:text-slate-700"
          >
            ← Volver
          </button>
        </div>
      </div>
    )
  }

  // --- Paso 1: login/registro normal ---
  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-100 px-4">
      <div className="w-full max-w-sm bg-white rounded-xl shadow p-8">
        <h1 className="text-2xl font-bold text-slate-800 mb-1">
          {isRegister ? 'Crear cuenta' : 'Iniciar sesión'}
        </h1>
        <p className="text-slate-500 text-sm mb-6">Banco Simplificado</p>

        {infoMessage && (
          <div className="text-sm text-blue-700 bg-blue-50 border border-blue-200 rounded-lg px-3 py-2 mb-4">
            {infoMessage}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          {isRegister && (
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Nombre completo
              </label>
              <input
                type="text"
                required
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Email</label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Contraseña</label>
            <input
              type="password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          {error && (
            <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white font-medium py-2 rounded-lg transition"
          >
            {loading ? 'Procesando...' : isRegister ? 'Registrarme' : 'Entrar'}
          </button>
        </form>

        <p className="text-sm text-slate-500 mt-6 text-center">
          {isRegister ? (
            <>
              ¿Ya tienes cuenta?{' '}
              <Link to="/login" className="text-blue-600 font-medium">
                Inicia sesión
              </Link>
            </>
          ) : (
            <>
              ¿No tienes cuenta?{' '}
              <Link to="/register" className="text-blue-600 font-medium">
                Regístrate
              </Link>
            </>
          )}
        </p>
      </div>
    </div>
  )
}
