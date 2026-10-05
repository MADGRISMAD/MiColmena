# MiColmena

Plataforma de gestión de tickets de soporte y atención al cliente.

## Estructura

```
MiColmena/
├── frontend/                    # Aplicación web (Vue)
│   ├── public/                  # Archivos servidos tal cual (favicon)
│   ├── src/
│   │   ├── assets/
│   │   │   ├── icons/           # Iconos del menú lateral
│   │   │   └── images/          # Logo y otras imágenes
│   │   ├── components/
│   │   │   └── layout/          # AppNavbar, AppSidebar
│   │   ├── router/              # Rutas de la aplicación
│   │   ├── services/            # Cliente de la API (api.js)
│   │   ├── styles/              # CSS global (Tailwind)
│   │   ├── views/               # Páginas: HomeView, DashboardView
│   │   ├── App.vue
│   │   └── main.js
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
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
| Frontend | Vue 3, Vue Router, Vite, Tailwind CSS |
| Backend | Go (`net/http` de la biblioteca estándar), `pgx` |
| Base de datos | PostgreSQL 16 |
| Autenticación | JWT + contraseñas con bcrypt |

## Levantar el frontend

```bash
cd frontend
npm install
npm run dev
```

## Levantar el backend

Con Docker (PostgreSQL + API):

```bash
echo "JWT_SECRET=$(openssl rand -hex 32)" > .env
echo "ADMIN_EMAIL=admin@micolmena.local" >> .env
echo "ADMIN_PASSWORD=una-contraseña-segura" >> .env
docker compose up -d
```

Sin Docker (con PostgreSQL ya instalado):

```bash
cd backend
cp .env.example .env   # y edita los valores
set -a; . ./.env; set +a
go run ./cmd/api
```

Las migraciones se aplican solas al arrancar. Con el backend en `:8080`, `npm run dev`
(en `frontend/`) redirige `/api` hacia él.

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

## Rendimiento

- Las listas usan paginación por cursor (`before`), que sigue siendo rápida con millones de filas.
- Los filtros habituales tienen índices, y la búsqueda usa texto completo de PostgreSQL en español.
- Solicitante, agente y autor se traen con `JOIN` en la misma consulta, sin N+1.
