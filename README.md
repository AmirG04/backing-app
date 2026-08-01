# 🏦 Online Banking System

Prueba técnica desarrollada como un sistema de banca en línea simplificado.

El proyecto permite a los usuarios autenticarse, administrar una cuenta bancaria y realizar operaciones financieras utilizando TigerBeetle como motor contable y PostgreSQL para la gestión de usuarios. Además, integra un chat con Inteligencia Artificial mediante Model Context Protocol (MCP), que permite realizar operaciones bancarias utilizando lenguaje natural, como consultar el saldo, revisar el historial de transacciones, realizar depósitos, retiros y transferencias. Para garantizar la seguridad, la IA solicita confirmación antes de ejecutar cualquier operación crítica.

---

## 🚀 Tecnologías

### Backend

- Go
- chi (router HTTP)
- pgx (driver PostgreSQL)
- JWT (golang-jwt)
- bcrypt

### Frontend

- React
- Vite
- JavaScript
- Tailwind CSS
- React Router

### Base de datos

- TigerBeetle
- PostgreSQL

### Infraestructura

- Docker
- Docker Compose
- GitHub

---

## 📂 Estructura del proyecto

```
banking-app/
├── backend/
│   ├── cmd/api/              # punto de entrada (main.go)
│   └── internal/
│       ├── config/
│       ├── db/                # Postgres + TigerBeetle
│       ├── handlers/          # auth, accounts, transactions, chat
│       ├── middleware/        # JWT
│       └── models/
├── frontend/
│   └── src/
│       ├── components/        # ChatWidget
│       ├── lib/                # api.js, auth.jsx
│       └── pages/              # Login, Dashboard, History
├── docs/
│   ├── API.md
│   ├── DATABASE.md
│   ├── DECISIONS.md
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

---

## 📌 Funcionalidades

### Autenticación

- Registro de usuarios
- Inicio de sesión
- JWT
- Cierre de sesión

### Cuentas

- Creación automática de cuenta bancaria
- Consulta de saldo
- Información de la cuenta

### Transacciones

- Depósitos
- Retiros
- Transferencias
- Historial de movimientos

### Dashboard

- Información general
- Últimas transacciones
- Chat integrado

### IA

> ⚠️ **Estado actual: en desarrollo.** La UI del chat ya está integrada en
> el dashboard y conectada a `/api/chat`, pero la integración con el
> modelo de IA vía MCP todavía no está implementada del lado del backend.

- Chat mediante lenguaje natural
- Confirmación antes de ejecutar operaciones críticas
- Consulta de saldo y movimientos

---

## 🏗 Arquitectura

- React consume una API REST desarrollada en Go.
- PostgreSQL almacena usuarios y autenticación.
- TigerBeetle administra todas las operaciones financieras.
- Docker Compose orquesta todos los servicios.

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

La documentación técnica se encuentra en la carpeta **docs**.

- API.md
- DATABASE.md
- DECISIONS.md
- KNOWN_ISSUES.md

---

## 👨‍💻 Autor

**Amir Gonzalez**

Prueba Técnica - 2026