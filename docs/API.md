# Documentación de la API

URL base (local): `http://localhost:8080`

## Autenticación

Todos los endpoints bajo `/api/` **excepto** `/api/auth/*` y `/health`
requieren un JWT en el header:

```
Authorization: Bearer <token>
```

El token se obtiene al hacer `register` o `login`, y expira a las 24 horas.

---

## Health check

### `GET /health`

Sin autenticación. Verifica que el servidor esté corriendo.

**Respuesta 200**
```json
{ "status": "ok" }
```

---

## Autenticación

### `POST /api/auth/register`

Crea un usuario nuevo y su cuenta bancaria principal.

**Body**
```json
{
  "email": "test@test.com",
  "password": "password123",
  "full_name": "Usuario Test"
}
```
`password` debe tener al menos 8 caracteres.

**Respuesta 201**
```json
{
  "token": "eyJhbGciOi...",
  "user": {
    "id": "b9a13e7f-...",
    "email": "test@test.com",
    "full_name": "Usuario Test",
    "created_at": "2026-08-01T21:03:12Z"
  },
  "account": {
    "id": "5b2f...",
    "tigerbeetle_account_id": "19fbf235...",
    "account_number": "4001-6588-5247-0001",
    "account_type": "checking",
    "currency": "USD",
    "created_at": "2026-08-01T21:03:12Z"
  }
}
```

**Errores**
| Código | Causa |
|---|---|
| 400 | Faltan campos, o password < 8 caracteres |
| 409 | El email ya está registrado |
| 500 | Error interno (Postgres/TigerBeetle) |

---

### `POST /api/auth/login`

**Body**
```json
{ "email": "test@test.com", "password": "password123" }
```

**Respuesta 200**: misma forma que `register` (token + user + account, con la cuenta principal del usuario).

**Errores**
| Código | Causa |
|---|---|
| 401 | Email o contraseña incorrectos |

---

### `POST /api/auth/logout`

Con JWT sin estado, el "cierre de sesión" real ocurre en el cliente
(borrando el token guardado). Este endpoint existe por completitud de la
API y como punto de extensión futuro (ej. blacklist de tokens).

**Respuesta 200**
```json
{ "message": "sesion cerrada" }
```

---

## Cuentas
*(requieren `Authorization: Bearer <token>`)*

### `GET /api/accounts`

Lista todas las cuentas del usuario autenticado (un usuario puede tener
varias), cada una con su saldo actual.

**Respuesta 200**
```json
[
  {
    "id": "5b2f...",
    "tigerbeetle_account_id": "19fbf235...",
    "account_number": "4001-6588-5247-0001",
    "account_type": "checking",
    "currency": "USD",
    "balance": 150,
    "created_at": "2026-08-01T21:03:12Z"
  }
]
```

---

### `GET /api/accounts/me`

Información del usuario autenticado (no de una cuenta específica).

**Respuesta 200**
```json
{
  "id": "b9a13e7f-...",
  "email": "test@test.com",
  "full_name": "Usuario Test",
  "created_at": "2026-08-01T21:03:12Z"
}
```

---

### `GET /api/accounts/balance`

Consulta el saldo de una cuenta.

**Query params**
- `account_id` (opcional): UUID de una cuenta específica (de la lista de
  `GET /api/accounts`). Si se omite, usa la cuenta principal (la más
  antigua del usuario).

**Respuesta 200**
```json
{ "account_number": "4001-6588-5247-0001", "balance": 150 }
```

---

## Transacciones
*(requieren `Authorization: Bearer <token>`; todas aceptan `?account_id=` opcional, igual que `/accounts/balance`)*

### `POST /api/transactions/deposit`

**Body**
```json
{ "amount": 100 }
```

**Respuesta 200**
```json
{ "message": "deposito realizado" }
```

**Errores**: `400` monto inválido · `422` no se pudo procesar

---

### `POST /api/transactions/withdraw`

**Body**
```json
{ "amount": 50 }
```

**Respuesta 200**
```json
{ "message": "retiro realizado" }
```

**Errores**: `400` monto inválido · `422 fondos insuficientes` si el saldo no alcanza

---

### `POST /api/transactions/transfer`

`to_account_id` es el **número de cuenta público** de la cuenta destino
(formato `4001-XXXX-XXXX-NNNN`, visible en `GET /api/accounts`) — no un ID
interno.

**Body**
```json
{ "to_account_id": "4001-1428-2372-0002", "amount": 25 }
```

**Respuesta 200**
```json
{ "message": "transferencia realizada" }
```

**Errores**: `400` monto inválido o transferencia a la propia cuenta · `404` la cuenta destino no existe · `422 fondos insuficientes`

---

### `GET /api/transactions/history`

Últimos 50 movimientos de la cuenta (depósitos, retiros y transferencias).

**Respuesta 200**
```json
[
  {
    "id": "e1f9...",
    "type": "transfer_sent",
    "amount": 25,
    "counterparty_account_number": "4001-1428-2372-0002",
    "timestamp": "2026-08-01T21:15:05Z"
  },
  {
    "id": "f800...",
    "type": "deposit",
    "amount": 100,
    "timestamp": "2026-08-01T21:10:00Z"
  }
]
```
`type` es uno de: `deposit`, `withdrawal`, `transfer_sent`, `transfer_received`.
`counterparty_account_number` solo aparece en transferencias (no en depósitos/retiros, cuya contraparte es el banco).

---

## Chat con IA (MCP)
*(requiere `Authorization: Bearer <token>`)*

### `POST /api/chat`

Recibe un mensaje en lenguaje natural. Internamente, el modelo de IA
decide qué operación bancaria realizar (si alguna) usando un servidor MCP
que expone las mismas operaciones que los endpoints REST de arriba.

**Body**
```json
{ "message": "¿cuánto dinero tengo?" }
```

**Respuesta 200**
```json
{
  "reply": "Tu saldo disponible es de **$150**.",
  "action_executed": true,
  "requires_confirmation": false
}
```

- `action_executed`: `true` si se ejecutó una operación real (depósito, retiro, transferencia).
- `requires_confirmation`: `true` cuando la IA identificó una operación **crítica** (retiro o transferencia) y está esperando que el usuario la confirme en su siguiente mensaje ("sí"/"no").

**Ejemplo de flujo de confirmación:**

1. Usuario: `"retira 50"` → respuesta con `requires_confirmation: true`, pidiendo confirmar.
2. Usuario: `"sí"` → respuesta con `action_executed: true`, el retiro ya se ejecutó.

Nota: el chat opera siempre sobre la cuenta principal del usuario (no
respeta el selector de cuenta del dashboard si el usuario tiene varias).

**Errores**: `400` mensaje vacío · `500` error consultando el modelo de IA (incluye cuando falta `OPENROUTER_API_KEY`)
