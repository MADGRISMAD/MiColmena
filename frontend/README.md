# MiColmena · Frontend

Aplicación web de MiColmena: SvelteKit + TypeScript + Tailwind CSS + shadcn-svelte.
Se compila a archivos estáticos (SPA) que hablan con la API en Go de `../backend`.

## Requisitos

- Node.js 22 o superior.
- **npm 11 o superior.** El npm 10 que trae Node 22 falla al resolver las dependencias;
  actualízalo con `npm install -g npm@11` (Node 24 ya trae npm 11).

## Comandos

```bash
npm install          # instalar dependencias
npm run dev          # servidor de desarrollo en http://localhost:5173
npm run build        # compilar a build/
npm run preview      # servir la compilación
npm run check        # comprobar tipos
npm run lint         # Prettier + ESLint
npm run test:unit    # pruebas unitarias (Vitest)
npm run test:e2e     # pruebas E2E (Playwright)
```

En desarrollo, las llamadas a `/api` se redirigen al backend en `http://localhost:8080`
(ver `vite.config.ts`). En producción, define `VITE_API_URL` con la URL del backend al compilar:

```bash
VITE_API_URL=https://api.micolmena.com npm run build
```

El contenido de `build/` se puede publicar en cualquier hosting estático (Cloudflare Pages,
Vercel, Netlify, Nginx…). Todas las rutas deben servir `index.html`.

## Estructura

```
src/
├── lib/
│   ├── api/              # Cliente tipado de la API y tipos de sus respuestas
│   ├── components/
│   │   ├── ui/           # Componentes de shadcn-svelte
│   │   └── *.svelte      # Componentes propios (badges, campos de formulario…)
│   ├── stores/auth.ts    # Sesión: token y usuario (Svelte stores)
│   ├── schemas.ts        # Validación de formularios (Zod)
│   ├── forms.ts          # Superforms en modo SPA conectado a la API
│   ├── format.ts         # Textos y fechas en español
│   └── navigation.ts
└── routes/
    ├── +page.svelte      # Portada
    ├── login/, register/
    └── (app)/            # Requiere sesión
        ├── dashboard/    # Solo agentes y administradores
        └── tickets/      # Lista, nuevo ticket y detalle
e2e/                      # Pruebas Playwright con la API simulada
```

## Decisiones

- **SPA estática (`adapter-static`, `ssr = false`).** El backend ya es la API en Go; así el
  frontend se publica gratis o casi gratis en un hosting estático.
- **Zod mini y `zod4Client`.** Validan igual que Zod normal pero pesan mucho menos, y evitan
  incluir en el navegador el generador de JSON Schema que Superforms solo necesita en servidor.
- **`tailwind-variants/lite`.** La combinación de clases ya la hace `cn`; la versión completa
  duplicaba ese código.
- **Formularios sin formsnap.** shadcn-svelte usa formsnap para su componente `Form`, pero
  formsnap aún no es compatible con Superforms 3 / SvelteKit 3. `lib/components/form-field.svelte`
  cubre lo mismo (etiqueta, error y atributos de accesibilidad).
- **Pruebas E2E sin backend.** `e2e/fake-api.ts` simula la API en el navegador, así corren en
  segundos y en CI sin Go ni PostgreSQL.

## shadcn-svelte

Para añadir componentes: `npx shadcn-svelte@latest add <componente>`. Usan el alias `#lib`
(SvelteKit 3 eliminó `$lib`), configurado en `components.json`.
