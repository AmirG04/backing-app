# 🏦 Online Banking System

Prueba técnica desarrollada como un sistema de banca en línea simplificado.

El proyecto permite a los usuarios autenticarse, administrar una o varias cuentas bancarias y realizar operaciones financieras utilizando TigerBeetle como motor contable y PostgreSQL para la gestión de usuarios. Además, integra un chat con Inteligencia Artificial mediante Model Context Protocol (MCP), que permite realizar operaciones bancarias utilizando lenguaje natural, como consultar el saldo, revisar el historial de transacciones, realizar depósitos, retiros y transferencias. Para garantizar la seguridad, la IA solicita confirmación antes de ejecutar cualquier operación crítica.

---

## 🚀 Tecnologías

### Backend

- Go
- chi (router HTTP)
- pgx (driver PostgreSQL)
- JWT (golang-jwt)
- bcrypt
- pquerna/otp (TOTP para 2FA)
- SDK oficial de Model Context Protocol (`modelcontextprotocol/go-sdk`)
- OpenRouter (acceso al modelo de IA, compatible con Claude/GPT/otros)

### Frontend

- React
- Vite
- JavaScript
- Tailwind CSS
- React Router

### Base de datos

- TigerBeetle (cuentas, transferencias, balances)
- PostgreSQL (usuarios, autenticación, cuentas)

### Infraestructura

- Docker
- Docker Compose
- GitHub

---

## 📂 Estructura del proyecto

```
banking-app/
├── backend/
│   ├── cmd/
│   │   ├── api/               # servidor principal (main.go)
│   │   └── seed/               # script para cargar datos de prueba masivos
│   ├── testdata/                # dataset de prueba simplificado
│   └── internal/
│       ├── ai/                  # cliente de OpenRouter
│       ├── config/
│       ├── db/                  # Postgres + TigerBeetle + generador de account_number
│       ├── handlers/            # auth, accounts, transactions, chat
│       ├── mcp/                  # servidor MCP con las herramientas bancarias
│       ├── middleware/          # JWT
│       └── models/
├── frontend/
│   └── src/
│       ├── components/          # ChatWidget
│       ├── lib/                  # api.js, auth.jsx
│       └── pages/                # Login, Dashboard, History
├── docs/
│   ├── API.md
│   └── KNOWN_ISSUES.md
├── docker-compose.yml
├── README.md
└── .env
```

---

## ⚙️ Requisitos

Antes de ejecutar el proyecto debes tener instalado:

- Go
- Node.js (LTS)
- Docker Desktop
- Git

## 🔑 Variables de entorno

Copia `.env.example` a `.env` en la raíz del proyecto:

```bash
cp .env.example .env
```

**El proyecto completo funciona sin ninguna variable de entorno adicional**
(auth, cuentas, transacciones e historial no dependen de ninguna key
externa). La única variable opcional es `OPENROUTER_API_KEY`, necesaria
únicamente para el chat con IA (`POST /api/chat`). Sin ella, el resto de
la aplicación sigue funcionando normalmente; el chat responde con un error
claro indicando que falta configurar la key.

Para conseguir una key: [openrouter.ai](https://openrouter.ai) → crear
cuenta → API Keys. El modelo por defecto es `anthropic/claude-sonnet-4.5`;
se puede cambiar con la variable opcional `OPENROUTER_MODEL` a cualquier
modelo del catálogo de OpenRouter que soporte "tool calling".

---

## ▶️ Ejecución

Clonar el repositorio

```bash
git clone <url-del-repositorio>
```

Entrar al proyecto

```bash
cd banking-app
```

Levantar todos los servicios

```bash
docker compose up --build
```

### Cargar datos de prueba (opcional)

Además del registro normal, el proyecto incluye un script para cargar
usuarios/cuentas/transacciones de prueba en volumen (útil para pruebas de
carga o para tener datos realistas rápido):

```bash
docker compose run --rm -v "${PWD}/backend/testdata:/testdata" backend ./seed /testdata/datos-prueba-simplificado.json
```

---

## 📌 Funcionalidades

### Autenticación

- Registro de usuarios
- Inicio de sesión
- JWT
- Cierre de sesión
- **(Bonus) Autenticación en dos pasos (2FA/TOTP)** — códigos de 6 dígitos
  compatibles con Google Authenticator/Authy. Opcional, se activa desde
  "Seguridad" en el dashboard. Cuando está activo, el login queda en un
  estado intermedio (token de pre-autenticación de 5 minutos, que no
  sirve para nada más) hasta que se confirma el código.

### Cuentas

- Un usuario puede tener **una o varias cuentas bancarias**
- Cada cuenta tiene un **número de cuenta público** (formato `4001-XXXX-XXXX-NNNN`),
  generado automáticamente al crearse - es el identificador que se usa
  para recibir transferencias (el ID interno de TigerBeetle nunca se expone)
- Consulta de saldo por cuenta
- Listado de todas las cuentas del usuario, con selector en el dashboard

### Transacciones

- Depósitos
- Retiros
- Transferencias (identificando la cuenta destino por su número de cuenta)
- Historial de movimientos, con el número de cuenta de la contraparte

### Dashboard

- Información general
- Selector de cuenta activa (si el usuario tiene más de una)
- Últimas transacciones
- Chat integrado

### IA (chat vía MCP)

- Chat mediante lenguaje natural, con confirmación obligatoria antes de
  ejecutar operaciones críticas (retiro/transferencia)
- Consulta de saldo, historial, depósitos, retiros y transferencias
- Arquitectura real de MCP: un servidor MCP (SDK oficial) expone las
  herramientas bancarias; el chat actúa como cliente MCP que las
  descubre (`tools/list`) y las invoca (`tools/call`) - no son llamadas
  directas a funciones Go
- El modelo de IA se accede vía OpenRouter (por defecto, Claude)

---

## 🏗 Arquitectura

- React consume una API REST desarrollada en Go.
- PostgreSQL almacena usuarios y sus cuentas (relación 1 usuario → N cuentas).
- TigerBeetle administra todas las operaciones financieras (balances y transferencias).
- El chat con IA usa un servidor MCP embebido (mismo proceso, protocolo real vía transporte en memoria) que expone las operaciones bancarias como herramientas, y un cliente que las conecta con un modelo de IA vía OpenRouter.
- Docker Compose orquesta los 5 servicios: postgres, tigerbeetle, backend, frontend.

---

## 🛠 Problemas conocidos y soluciones

Durante el desarrollo se presentaron varios problemas de infraestructura al
correr TigerBeetle dentro de Docker. Resumen:

- **Backend en loop de reinicio (`io_uring is not available`):** el
  cliente de TigerBeetle necesita `security_opt: seccomp:unconfined`,
  `cap_add: IPC_LOCK` y `ulimits.memlock` habilitados en **cualquier**
  contenedor que lo use — no solo en el del servidor de TigerBeetle.
- **`Invalid client cluster address`:** el cliente de TigerBeetle no
  acepta nombres de host de Docker (ej. `tigerbeetle:3000`), solo IPs
  literales. Se resolvió asignando una IP fija al servicio `tigerbeetle`
  dentro de una red Docker propia.
- **CGO / compilación:** el cliente de TigerBeetle usa CGO, así que el
  Dockerfile del backend necesita `CGO_ENABLED=1`, un compilador de C, y
  una imagen base con `glibc` (no `musl`/Alpine).

Detalle completo, con síntomas y comandos exactos, en
[`docs/KNOWN_ISSUES.md`](./docs/KNOWN_ISSUES.md).

---

## 📖 Documentación

- [`docs/API.md`](./docs/API.md) — documentación de todos los endpoints (auth, cuentas, transacciones, chat), con ejemplos de request/response y códigos de error.
- [`docs/KNOWN_ISSUES.md`](./docs/KNOWN_ISSUES.md) — problemas de infraestructura encontrados y su solución.

---

## 👨‍💻 Autor

**Amir Gonzalez**

Prueba Técnica - 2026
