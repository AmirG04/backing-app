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

  async function login(email, password) {
    const data = await api.login({ email, password })
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
    localStorage.setItem('account', JSON.stringify(data.account))
    setUser(data.user)
    setAccount(data.account)
  }

  async function register(email, password, full_name) {
    const data = await api.register({ email, password, full_name })
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
    localStorage.setItem('account', JSON.stringify(data.account))
    setUser(data.user)
    setAccount(data.account)
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
    <AuthContext.Provider value={{ user, account, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  return useContext(AuthContext)
}
