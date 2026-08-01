# 🏦 Online Banking System

Prueba técnica desarrollada como un sistema de banca en línea simplificado.

El proyecto permite a los usuarios autenticarse, administrar una cuenta bancaria y realizar operaciones financieras utilizando TigerBeetle como motor contable y PostgreSQL para la gestión de usuarios. Además, integra un chat con Inteligencia Artificial mediante Model Context Protocol (MCP), que permite realizar operaciones bancarias utilizando lenguaje natural, como consultar el saldo, revisar el historial de transacciones, realizar depósitos, retiros y transferencias. Para garantizar la seguridad, la IA solicita confirmación antes de ejecutar cualquier operación crítica.

---

## 🚀 Tecnologías

### Backend

- Go
- Gin
- JWT
- bcrypt

### Frontend

- React
- Vite
- TypeScript
- Tailwind CSS

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
online-banking/
├── backend/
├── frontend/
├── docs/
├── docker-compose.yml
├── README.md
└── .env.example
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
cd online-banking
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