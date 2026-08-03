const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

function getToken() {
  return localStorage.getItem('token')
}

async function request(path, options = {}) {
  const token = getToken()
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers || {}),
  }

  const res = await fetch(`${API_URL}${path}`, { ...options, headers })
  const data = await res.json().catch(() => ({}))

  if (!res.ok) {
    throw new Error(data.error || `Error ${res.status}`)
  }
  return data
}

// Agrega ?account_id=... a una ruta si se especifica una cuenta activa
// (si no, el backend usa la cuenta principal del usuario por defecto).
function withAccount(path, accountId) {
  return accountId ? `${path}?account_id=${encodeURIComponent(accountId)}` : path
}

export const api = {
  register: (payload) =>
    request('/api/auth/register', { method: 'POST', body: JSON.stringify(payload) }),
  login: (payload) =>
    request('/api/auth/login', { method: 'POST', body: JSON.stringify(payload) }),
  logout: () => request('/api/auth/logout', { method: 'POST' }),
  getAccountInfo: () => request('/api/accounts/me'),
  getAccounts: () => request('/api/accounts'),
  getBalance: (accountId) => request(withAccount('/api/accounts/balance', accountId)),
  deposit: (amount, accountId) =>
    request(withAccount('/api/transactions/deposit', accountId), {
      method: 'POST',
      body: JSON.stringify({ amount }),
    }),
  withdraw: (amount, accountId) =>
    request(withAccount('/api/transactions/withdraw', accountId), {
      method: 'POST',
      body: JSON.stringify({ amount }),
    }),
  transfer: (to_account_id, amount, accountId) =>
    request(withAccount('/api/transactions/transfer', accountId), {
      method: 'POST',
      body: JSON.stringify({ to_account_id, amount }),
    }),
  history: (accountId) => request(withAccount('/api/transactions/history', accountId)),
}
