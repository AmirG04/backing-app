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

export const api = {
  register: (payload) =>
    request('/api/auth/register', { method: 'POST', body: JSON.stringify(payload) }),
  login: (payload) =>
    request('/api/auth/login', { method: 'POST', body: JSON.stringify(payload) }),
  logout: () => request('/api/auth/logout', { method: 'POST' }),
  getAccountInfo: () => request('/api/accounts/me'),
  getBalance: () => request('/api/accounts/balance'),
  deposit: (amount) =>
    request('/api/transactions/deposit', { method: 'POST', body: JSON.stringify({ amount }) }),
  withdraw: (amount) =>
    request('/api/transactions/withdraw', { method: 'POST', body: JSON.stringify({ amount }) }),
  transfer: (to_account_id, amount) =>
    request('/api/transactions/transfer', {
      method: 'POST',
      body: JSON.stringify({ to_account_id, amount }),
    }),
  history: () => request('/api/transactions/history'),
}
