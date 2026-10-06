# MiColmena

Plataforma de gestión de tickets de soporte y atención al cliente.

## Estructura

```
MiColmena/
├── frontend/                    # Aplicación web (SvelteKit)
│   ├── src/lib/                 # Cliente de la API, stores, esquemas, componentes
│   ├── src/routes/              # Páginas: portada, login, registro, dashboard, tickets
│   ├── e2e/                     # Pruebas Playwright
│   └── README.md                # Detalles del frontend
├── backend/                     # API (Go)
│   ├── cmd/api/                 # Punto de entrada del servidor
│   ├── internal/
│   │   ├── api/                 # Rutas y handlers HTTP
│   │   ├── auth/                # Contraseñas y tokens JWT
│   │   ├── config/              # Variables de entorno
│   │   └── db/                  # Conexión y migraciones SQL
│   ├── Dockerfile
│   └── go.mod
├── docker-compose.yml           # PostgreSQL + API
└── README.md
```

## Tecnologías

| Parte | Tecnología |
|---|---|
| Frontend | SvelteKit 3 (SPA estática), Svelte 5, TypeScript, Tailwind CSS 4, shadcn-svelte |
| Formularios y estado | Superforms + Zod, Svelte stores |
| Pruebas del frontend | Vitest (unitarias), Playwright (E2E) |
| Backend | Go (`net/http` de la biblioteca estándar), `pgx` |
| Base de datos | PostgreSQL 16 |
| Autenticación | JWT + contraseñas con bcrypt |

## Correr el proyecto en local

Necesitas **Node.js 22+ con npm 11+** (`npm install -g npm@11`; con el npm 10 que trae Node 22 la instalación falla), **Go 1.26** (Go descarga solo la versión correcta si tienes 1.21 o más) y **Docker** o un PostgreSQL propio.

Se usan tres terminales:

**1. Base de datos**

```bash
docker compose up -d db
```

Crea PostgreSQL con el usuario, la contraseña y la base `micolmena`, que coinciden con `backend/.env.example`. Sin Docker, crea tú ese usuario y esa base en tu PostgreSQL, o cambia `DATABASE_URL` en `backend/.env`.

**2. API** (en `http://localhost:8080`)

```bash
cd backend
cp .env.example .env
set -a; . ./.env; set +a     # Linux, macOS, WSL o Git Bash
go run ./cmd/api
```

Aplica las migraciones solo y crea un administrador: `admin@micolmena.local` con la contraseña `cambia-esta-contraseña` (se cambian en `.env`).

**3. Frontend** (en `http://localhost:5173`)

```bash
cd frontend
npm install
npm run dev
```

Abre http://localhost:5173. Las llamadas a `/api` se redirigen solas al backend en `:8080`. Inicia sesión con el administrador, que ve todo igual que un agente. Para probar la vista de cliente, crea otra cuenta desde "Crear cuenta". Convertir a alguien en agente se hace por ahora con la API (`PATCH /api/users/{id}/role`, como administrador).

**Si te bloquea el login mientras pruebas:** son los límites contra fuerza bruta (ver más abajo). Reinicia la API para borrar el bloqueo, o sube `AUTH_RATE_PER_MIN` y `LOGIN_MAX_FAILURES` en `backend/.env`.

### Todo en Docker

```bash
echo "JWT_SECRET=$(openssl rand -hex 32)" > .env
echo "ADMIN_EMAIL=admin@micolmena.local" >> .env
echo "ADMIN_PASSWORD=una-contraseña-segura" >> .env
docker compose up -d
```

Levanta PostgreSQL y la API (`:8080`). El frontend se corre aparte con `npm run dev`, como arriba. Más detalles en [`frontend/README.md`](frontend/README.md).

## API

Todas las rutas, salvo `health`, `register` y `login`, requieren `Authorization: Bearer <token>`.

| Método | Ruta | Quién | Descripción |
|---|---|---|---|
| GET | `/api/health` | todos | Estado del servicio y la base de datos |
| POST | `/api/auth/register` | todos | Crea una cuenta de cliente |
| POST | `/api/auth/login` | todos | Devuelve un token |
| GET | `/api/me` | autenticado | Usuario actual |
| GET | `/api/tickets` | autenticado | Lista con filtros `status`, `priority`, `assignee` (`me`, `none` o id), `q` (búsqueda), `before` (cursor), `limit` |
| POST | `/api/tickets` | autenticado | Crea un ticket |
| GET | `/api/tickets/{id}` | autenticado | Detalle |
| PATCH | `/api/tickets/{id}` | autenticado | Edita; los clientes solo cambian título y descripción, o cierran |
| GET | `/api/tickets/{id}/comments` | autenticado | Comentarios (los clientes no ven las notas internas) |
| POST | `/api/tickets/{id}/comments` | autenticado | Agrega un comentario o una nota interna |
| GET | `/api/users?role=agent` | agentes | Lista de usuarios |
| PATCH | `/api/users/{id}/role` | admin | Cambia el rol (`admin`, `agent`, `customer`) |
| GET | `/api/stats` | agentes | Números del dashboard |

Los clientes solo ven sus propios tickets.

## Protección contra fuerza bruta

`/api/auth/login` y `/api/auth/register` tienen dos límites. Al superarlos, la API responde `429 Too Many Requests` con la cabecera `Retry-After` (segundos).

| Límite | Por defecto | Variable |
|---|---|---|
| Intentos por minuto y por IP (login y registro comparten el contador) | 10 | `AUTH_RATE_PER_MIN` |
| Fallos de contraseña seguidos de un mismo email | 5 | `LOGIN_MAX_FAILURES` |
| Duración del bloqueo de ese email | 15 min | `LOGIN_LOCKOUT` |

Mientras un email está bloqueado, ni la contraseña correcta entra. Un login correcto reinicia su contador.

**Si pones la API detrás de un proxy (Caddy, Nginx…), activa `TRUST_PROXY=true`.** Así se usa la IP real del cliente, que el proxy añade al final de `X-Forwarded-For`. Sin proxy déjalo en `false`: si no, cualquiera podría falsificar esa cabecera para esquivar el límite.

Límites a tener en cuenta:
- **El estado vive en la memoria de la API.** Sirve para una sola instancia; con varias habría que moverlo a Redis o PostgreSQL. Reiniciar la API lo borra.
- **Bloquear por email permite molestar a alguien:** quien conozca tu email puede provocar un bloqueo de 15 minutos enviando contraseñas incorrectas. Es el costo de frenar la fuerza bruta; el bloqueo expira solo y no revela si la cuenta existe.
- Las IPv6 se agrupan por bloque /64, porque una sola persona suele tener un bloque entero.

## Pruebas

```bash
# Backend. Las pruebas de la API y la base de datos necesitan un PostgreSQL:
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
cd backend && go test -race ./...

# Frontend
cd frontend && npm run check && npm run lint && npm run test:unit -- --run && npm run test:e2e
```

Cada prueba del backend crea su propio esquema de PostgreSQL y lo borra al terminar, así que no pisan los datos de esa base ni se pisan entre sí. Sin `TEST_DATABASE_URL`, esas pruebas se saltan y el resto corre normal.

GitHub Actions ejecuta todo esto en cada push a `main` y en cada pull request (`.github/workflows/ci.yml`): formato, `go vet`, pruebas con `-race` contra PostgreSQL 16, construcción de la imagen Docker, y en el frontend tipos, lint, pruebas unitarias y E2E.

## Rendimiento

- Las listas usan paginación por cursor (`before`), que sigue siendo rápida con millones de filas.
- Los filtros habituales tienen índices, y la búsqueda usa texto completo de PostgreSQL en español.
- Solicitante, agente y autor se traen con `JOIN` en la misma consulta, sin N+1.
