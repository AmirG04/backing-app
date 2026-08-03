const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// sessionStorage (no localStorage) a proposito: la sesion no debe
// sobrevivir a cerrar el navegador/pestaña ni a reiniciar la computadora.
function getToken() {
  return sessionStorage.getItem('token')
}

function forceLogout() {
  sessionStorage.removeItem('token')
  sessionStorage.removeItem('user')
  sessionStorage.removeItem('account')
  if (!window.location.pathname.startsWith('/login')) {
    window.location.href = '/login'
  }
}

// tokenOverride permite mandar un token distinto al guardado en sesion
// (se usa solo para completar el 2FA, que manda el pre_auth_token).
async function request(path, options = {}, tokenOverride) {
  const token = tokenOverride || getToken()
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers || {}),
  }

  const res = await fetch(`${API_URL}${path}`, { ...options, headers })
  const data = await res.json().catch(() => ({}))

  // Si el token de sesion normal fue rechazado (expirado/invalido), forzamos
  // el logout - pero NO cuando se uso un tokenOverride (ej. el 2FA), donde
  // un 401 solo significa "codigo incorrecto", no una sesion muerta.
  if (res.status === 401 && !tokenOverride) {
    forceLogout()
  }

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
  login2FA: (preAuthToken, code) =>
    request(
      '/api/auth/2fa/login',
      { method: 'POST', body: JSON.stringify({ code }) },
      preAuthToken
    ),
  logout: () => request('/api/auth/logout', { method: 'POST' }),
  getAccountInfo: () => request('/api/accounts/me'),
  getAccounts: () => request('/api/accounts'),
  createAccount: (accountType) =>
    request('/api/accounts', { method: 'POST', body: JSON.stringify({ account_type: accountType }) }),
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

  // 2FA
  setup2FA: () => request('/api/2fa/setup', { method: 'POST' }),
  verify2FA: (code) =>
    request('/api/2fa/verify', { method: 'POST', body: JSON.stringify({ code }) }),
  disable2FA: (password) =>
    request('/api/2fa/disable', { method: 'POST', body: JSON.stringify({ password }) }),
}
