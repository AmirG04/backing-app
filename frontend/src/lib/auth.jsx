import React, { createContext, useContext, useState } from 'react'
import { api } from './api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    const stored = localStorage.getItem('user')
    return stored ? JSON.parse(stored) : null
  })
  const [account, setAccount] = useState(() => {
    const stored = localStorage.getItem('account')
    return stored ? JSON.parse(stored) : null
  })

  function persistSession(data) {
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
    localStorage.setItem('account', JSON.stringify(data.account))
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
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    localStorage.removeItem('account')
    setUser(null)
    setAccount(null)
  }

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
