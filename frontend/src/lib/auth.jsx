import React, { createContext, useContext, useEffect, useState } from 'react'
import { api } from './api'

const AuthContext = createContext(null)

// Tiempo de inactividad antes de cerrar sesion sola.
const INACTIVITY_TIMEOUT_MS = 15 * 60 * 1000
const ACTIVITY_EVENTS = ['mousemove', 'keydown', 'click', 'scroll', 'touchstart']

export function AuthProvider({ children }) {
  // sessionStorage (no localStorage) a proposito: la sesion no debe
  // sobrevivir a cerrar el navegador/pestaña ni a reiniciar la computadora.
  const [user, setUser] = useState(() => {
    const stored = sessionStorage.getItem('user')
    return stored ? JSON.parse(stored) : null
  })
  const [account, setAccount] = useState(() => {
    const stored = sessionStorage.getItem('account')
    return stored ? JSON.parse(stored) : null
  })

  function persistSession(data) {
    sessionStorage.setItem('token', data.token)
    sessionStorage.setItem('user', JSON.stringify(data.user))
    sessionStorage.setItem('account', JSON.stringify(data.account))
    setUser(data.user)
    setAccount(data.account)
  }

  // Devuelve { requiresTwoFactor: true, preAuthToken } si el usuario tiene
  // 2FA activo (todavia no inicia sesion), o simplemente inicia sesion si no.
  async function login(email, password) {
    const data = await api.login({ email, password })
    if (data.requires_2fa) {
      return { requiresTwoFactor: true, preAuthToken: data.pre_auth_token }
    }
    persistSession(data)
    return { requiresTwoFactor: false }
  }

  // Segundo paso del login cuando hay 2FA activo.
  async function completeTwoFactorLogin(preAuthToken, code) {
    const data = await api.login2FA(preAuthToken, code)
    persistSession(data)
  }

  async function register(email, password, full_name) {
    const data = await api.register({ email, password, full_name })
    persistSession(data)
  }

  function logout() {
    api.logout().catch(() => {})
    sessionStorage.removeItem('token')
    sessionStorage.removeItem('user')
    sessionStorage.removeItem('account')
    setUser(null)
    setAccount(null)
  }

  // Cierra la sesion sola si no hay actividad del usuario (mouse, teclado,
  // clics, scroll) durante INACTIVITY_TIMEOUT_MS. Solo corre mientras hay
  // una sesion activa.
  useEffect(() => {
    if (!user) return

    let timer
    let lastReset = 0

    function handleInactivityTimeout() {
      sessionStorage.setItem('logout_reason', 'inactivity')
      logout()
    }

    function resetTimer() {
      const now = Date.now()
      // throttle: no reiniciar el timer mas de una vez cada 5s, no hace
      // falta reaccionar a cada pixel de movimiento del mouse.
      if (now - lastReset < 5000) return
      lastReset = now
      clearTimeout(timer)
      timer = setTimeout(handleInactivityTimeout, INACTIVITY_TIMEOUT_MS)
    }

    resetTimer()
    ACTIVITY_EVENTS.forEach((evt) => window.addEventListener(evt, resetTimer))

    return () => {
      clearTimeout(timer)
      ACTIVITY_EVENTS.forEach((evt) => window.removeEventListener(evt, resetTimer))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user])

  return (
    <AuthContext.Provider
      value={{ user, account, login, completeTwoFactorLogin, register, logout }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  return useContext(AuthContext)
}
