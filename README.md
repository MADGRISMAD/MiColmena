# MiColmena

Plataforma de gestión de tickets de soporte y atención al cliente.

## Funciones

- **Tickets** con estados, prioridades, categorías, etiquetas y búsqueda en español.
- **Conversación** con respuestas públicas, notas internas, menciones `@Nombre`, adjuntos e historial de cambios.
- **Respuestas guardadas** con variables (`{{solicitante}}`, `{{agente}}`, `{{ticket}}`) que también pueden cambiar el estado.
- **Acciones masivas** (estado, prioridad, asignación, etiquetas) y **vistas guardadas** por usuario.
- **SLA** por prioridad: plazos de primera respuesta y resolución, aviso de riesgo y vista de vencidos.
- **Encuesta de satisfacción** al resolver.
- **Notificaciones** en la app (campana) y por correo, y **tiempo real**: lo que otro cambia se ve sin recargar.
- **Centro de ayuda** público con buscador; al abrir un ticket se sugieren artículos.
- **Reportes** por periodo (volumen, tiempos, SLA, satisfacción, por agente y categoría) y **exportación a CSV**.
- **Administración**: alta de agentes, roles, desactivar cuentas, categorías y plazos de SLA.
- **Cuenta**: perfil, cambio de contraseña (cierra las demás sesiones) y recuperación por correo.

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
| POST | `/api/auth/register` · `login` | todos | Crea una cuenta de cliente / devuelve un token |
| POST | `/api/auth/forgot` · `reset` | todos | Envía el enlace de recuperación / cambia la contraseña con él |
| GET, PATCH | `/api/me` | autenticado | Usuario actual; cambiar nombre, email (pide la contraseña) y avisos por correo |
| POST | `/api/me/password` | autenticado | Cambia la contraseña y devuelve un token nuevo |
| GET | `/api/tickets` | autenticado | Lista con filtros `status`, `priority`, `assignee` (`me`, `none` o id), `tag`, `sla=breached`, `q`, `before` (cursor), `limit` |
| POST | `/api/tickets` | autenticado | Crea un ticket |
| GET, PATCH | `/api/tickets/{id}` | autenticado | Detalle y edición; los clientes solo cambian título y descripción, o cierran |
| POST | `/api/tickets/bulk` | agentes | Mismo cambio a varios tickets |
| GET, POST | `/api/tickets/{id}/comments` | autenticado | Comentarios (los clientes no ven las notas internas) |
| GET | `/api/tickets/{id}/events` | autenticado | Historial de cambios |
| GET, POST | `/api/tickets/{id}/attachments` | autenticado | Adjuntos (multipart, campo `file` y opcional `comment_id`) |
| GET, DELETE | `/api/attachments/{id}` | autenticado | Descargar o borrar un adjunto |
| POST | `/api/tickets/{id}/satisfaction` | solicitante | Valoración `good` o `bad` con comentario |
| GET | `/api/notifications` · POST `/read` | autenticado | Notificaciones y marcarlas como leídas |
| GET | `/api/stream` | autenticado | Avisos en tiempo real (Server-Sent Events) |
| GET, POST, DELETE | `/api/views` | autenticado | Vistas guardadas |
| GET | `/api/users` | agentes | Usuarios (`role`, `q`, `active=all`) |
| POST, PATCH | `/api/users`, `/api/users/{id}` | admin | Crear usuarios; cambiar datos, rol, contraseña o desactivar |
| GET | `/api/stats` | agentes | Números del dashboard |
| GET | `/api/tags` | agentes | Etiquetas en uso |
| GET · POST, PATCH, DELETE | `/api/categories` | todos · admin | Categorías |
| GET, POST, PATCH, DELETE | `/api/macros` | agentes | Respuestas guardadas |
| GET · PUT | `/api/sla` | agentes · admin | Plazos por prioridad |
| GET | `/api/reports`, `/api/reports/tickets.csv` | agentes | Reporte por rango (`from`, `to`, `tz`) y exportación |
| GET · POST, PATCH, DELETE | `/api/articles` | todos · agentes | Centro de ayuda (sin sesión, solo los publicados) |
| POST · GET, PATCH | `/api/leads` | todos · admin | Solicitudes de demo desde la landing |

Los clientes solo ven sus propios tickets.

## Página principal y ventas

- Los **planes y precios** (en pesos, por agente al mes, más IVA) están en `frontend/src/lib/pricing.ts`. Cambia ahí precios, límites o lo que incluye cada plan; la landing, el formulario de demo y el panel de solicitudes los leen de ese archivo.
- El formulario **Solicitar demo** guarda la solicitud (`POST /api/leads`, con límite por IP y un campo trampa contra bots) y avisa a los administradores en la campana y por correo. Se gestionan en *Administración → Solicitudes de demo*.

## Correo y adjuntos

- **Correo.** Los avisos se guardan en una cola (`email_outbox`) dentro de la misma transacción y un proceso de la API los envía con reintentos. Configura `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD` y `SMTP_FROM` con los datos de tu proveedor (Brevo, Resend, Amazon SES…), y `APP_URL` con la dirección del frontend para los enlaces. **Sin `SMTP_HOST` los correos no se envían: se escriben en el log**, útil en desarrollo para copiar el enlace de recuperación. Para que no lleguen a spam necesitarás un dominio propio con SPF y DKIM.
- **Adjuntos.** Se guardan en disco en `UPLOAD_DIR` (en Docker, el volumen `uploads`) con un tamaño máximo de `MAX_UPLOAD_MB` (10 MB por defecto). Inclúyelos en las copias de seguridad junto con la base de datos.
- **Tiempo real.** Los avisos viven en la memoria de la API: con varias instancias, cada una solo avisa a sus conexiones. Si pones Nginx delante, desactiva el buffering para `/api/stream` (la API ya envía `X-Accel-Buffering: no`).

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
