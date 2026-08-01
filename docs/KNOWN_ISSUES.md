# Troubleshooting

## `backend` en loop de reinicio: `io_uring is not available` / `PermissionDenied`

### Síntoma

Al correr `docker compose up`, el contenedor `backend` queda en estado
`Restarting` y los logs muestran algo como:

```
error(io): io_uring is not available
error(io): likely cause: the syscall is disabled by sysctl, try 'sysctl -w kernel.io_uring_disabled=0'
error(tb_client_context): ...: failed to initialize IO: PermissionDenied
thread N panic: attempt to unwrap error: PermissionDenied
```

### Causa

El cliente oficial de TigerBeetle (usado dentro del backend vía CGO) depende
de `io_uring` para sus operaciones de red, igual que el propio servidor de
TigerBeetle. Docker bloquea las syscalls de `io_uring` por defecto en su
perfil de seguridad (`seccomp`), así que hay que habilitarlas explícitamente
por contenedor.

**Importante:** esto hay que configurarlo en *cada* contenedor que use el
cliente de TigerBeetle — no solo en el contenedor del servidor. Es un error
fácil de cometer, porque el servidor puede levantar perfectamente bien
mientras el backend (que también corre el cliente embebido) sigue fallando,
lo cual parece contradictorio a primera vista.

### Solución

En `docker-compose.yml`, cualquier servicio que use el cliente de
TigerBeetle necesita:

```yaml
security_opt:
  - seccomp:unconfined
cap_add:
  - IPC_LOCK
ulimits:
  memlock:
    soft: -1
    hard: -1
```

### Nota para Windows + Docker Desktop

Si después de agregar esta configuración el error persiste **exactamente
igual**, verifica que Docker Compose realmente esté aplicando el cambio al
contenedor:

```bash
docker inspect <nombre_del_contenedor> --format "{{.HostConfig.SecurityOpt}}"
```

Si el resultado es `[]` (vacío) en vez de `[seccomp:unconfined]`, el
contenedor no se recreó con la configuración nueva. Soluciónalo con:

```bash
docker compose down
docker compose up --build --force-recreate
```

Y vuelve a verificar con `docker inspect`.

---

## Direcciones de red de TigerBeetle: no acepta nombres de host de Docker

### Síntoma

```
fatal: error creando cliente tigerbeetle: Invalid client cluster address.
```

### Causa

El cliente de TigerBeetle valida la dirección en `--addresses` y **no
acepta nombres de dominio/host personalizados** (como el nombre de un
servicio de Docker, ej. `tigerbeetle:3000`) — solo IPs literales.

### Solución

Se creó una red Docker (`banking_net`) con un rango de IPs fijo, y se le
asignó al servicio `tigerbeetle` una IP estática (`172.28.0.10`). El backend
se conecta a esa IP directamente en vez del nombre del servicio:

```yaml
networks:
  banking_net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.28.0.0/16

services:
  tigerbeetle:
    networks:
      banking_net:
        ipv4_address: 172.28.0.10

  backend:
    environment:
      TIGERBEETLE_ADDRESS: 172.28.0.10:3000
```

---

## El cliente Go de TigerBeetle usa CGO

TigerBeetle no es una librería Go pura — usa CGO para enlazar con su cliente
nativo (`tb_client`), compilado en Zig. Esto tiene dos implicaciones:

1. **En Windows, `go build` local puede fallar** con errores de "build
   constraints exclude all Go files" a menos que tengas un compilador C
   instalado (Zig, según la documentación oficial de TigerBeetle). Esto no
   es necesario para el proyecto en sí — solo compila dentro de Docker
   (Linux), donde el `Dockerfile` ya instala las herramientas necesarias.

2. **El `Dockerfile` del backend necesita:**
   - `CGO_ENABLED=1` (no `0`)
   - Un compilador de C real (`build-essential` en la etapa de build)
   - Una imagen base con `glibc`, no `musl` (por eso se usa
     `golang:1.22-bookworm` / `debian:bookworm-slim` en vez de las
     variantes `-alpine`, que usan `musl` y suelen dar problemas con las
     librerías nativas precompiladas de TigerBeetle).