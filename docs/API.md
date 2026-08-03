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

**Respuesta 200 — si el usuario NO tiene 2FA activo**: misma forma que
`register` (`token` + `user` + `account`, con la cuenta principal).

**Respuesta 200 — si el usuario SÍ tiene 2FA activo**: no entrega un
token de sesión todavía, solo un token temporal de pre-autenticación:
```json
{
  "requires_2fa": true,
  "pre_auth_token": "eyJhbGciOi..."
}
```
Ese `pre_auth_token` expira en 5 minutos y **solo sirve** para completar
el segundo factor en `POST /api/auth/2fa/login` — no funciona en ningún
otro endpoint protegido (el middleware lo rechaza explícitamente).

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

## Autenticación en dos pasos (2FA / TOTP)

*(Bonus del enunciado: "Autenticación adicional")*

Códigos de 6 dígitos compatibles con Google Authenticator, Authy, etc.
2FA es **opcional** y lo activa cada usuario desde su cuenta ya logueada.

### `POST /api/2fa/setup`
*(requiere `Authorization: Bearer <token>` de sesión normal)*

Genera un secreto TOTP nuevo para el usuario y lo guarda como
"pendiente" (todavía no protege el login hasta confirmarse con
`/api/2fa/verify`). Si se llama de nuevo antes de confirmar, reemplaza
el secreto pendiente anterior.

**Respuesta 200**
```json
{
  "secret": "DZNBLWCMMQ6DXWOSBAWA3OFRLDS5XHT6",
  "otpauth_url": "otpauth://totp/Banco%20Simplificado:test@test.com?secret=...&issuer=Banco%20Simplificado"
}
```
El frontend muestra `secret` para ingresarlo manualmente en la app
autenticadora (o `otpauth_url` como link directo en móviles).

---

### `POST /api/2fa/verify`
*(requiere `Authorization: Bearer <token>` de sesión normal)*

Confirma la activación con el primer código generado por la app.

**Body**
```json
{ "code": "123456" }
```

**Respuesta 200**
```json
{ "message": "2FA activado correctamente" }
```

**Errores**: `400` no hay setup pendiente · `401` código inválido

---

### `POST /api/2fa/disable`
*(requiere `Authorization: Bearer <token>` de sesión normal)*

Requiere la contraseña actual (no solo estar logueado), ya que reduce la
protección de la cuenta.

**Body**
```json
{ "password": "password123" }
```

**Respuesta 200**
```json
{ "message": "2FA desactivado" }
```

**Errores**: `401` contraseña incorrecta

---

### `POST /api/auth/2fa/login`

Segundo paso del login cuando el usuario tiene 2FA activo. **No usa el
token de sesión normal** — usa el `pre_auth_token` que devolvió
`/api/auth/login`.

**Header**
```
Authorization: Bearer <pre_auth_token>
```

**Body**
```json
{ "code": "123456" }
```

**Respuesta 200**: token de sesión completo (`token` + `user` + `account`), igual que un login exitoso normal.

**Errores**: `401` token de pre-autenticación inválido/expirado, o código incorrecto

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

### `POST /api/accounts`

Crea una cuenta bancaria adicional para el usuario autenticado (ya
tiene una desde el registro; puede abrir más, ej. una de ahorro además
de la corriente).

**Body** (opcional; por defecto `checking`/`USD`)
```json
{ "account_type": "savings", "currency": "USD" }
```
`account_type` debe ser `"checking"` o `"savings"`.

**Respuesta 201**: un objeto `Account`, igual al de `GET /api/accounts` (sin `balance`, ya que arranca en 0).

**Errores**: `400` account_type inválido

---


### `GET /api/accounts/me`

Información del usuario autenticado (no de una cuenta específica).

**Respuesta 200**
```json
{
  "id": "b9a13e7f-...",
  "email": "test@test.com",
  "full_name": "Usuario Test",
  "two_factor_enabled": false,
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
{
  "message": "¿cuánto dinero tengo?",
  "history": [
    { "role": "user", "content": "hola" },
    { "role": "assistant", "content": "¡Hola! ¿En qué te ayudo?" }
  ]
}
```
`history` es opcional: turnos previos de la conversación (solo texto,
sin detalles de herramientas), para que el modelo tenga contexto de lo
ya hablado. El frontend lo arma automáticamente con los mensajes que
ya se ven en pantalla; el backend limita a los últimos 12 turnos.

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

**Múltiples cuentas:** si el usuario tiene más de una cuenta y no
especifica sobre cuál operar, el chat pregunta cuál usar (mencionando
número y tipo) en vez de adivinar - tanto para consultas como para
operaciones críticas, donde la cuenta se resuelve *antes* de pedir
confirmación.

**Errores**: `400` mensaje vacío · `500` error consultando el modelo de IA (incluye cuando falta `OPENROUTER_API_KEY`)
