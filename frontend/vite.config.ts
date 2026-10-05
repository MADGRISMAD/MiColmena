import { defineConfig } from 'vitest/config';
import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			// SPA estática: se sirve desde cualquier hosting de archivos (Cloudflare Pages, Vercel, Nginx…).
			// index.html atiende todas las rutas y el router del navegador resuelve el resto.
			adapter: adapter({ fallback: 'index.html' })
		})
	],
	server: {
		// En desarrollo, las llamadas a /api van al backend en Go.
		proxy: {
			'/api': 'http://localhost:8080'
		}
	},
	test: {
		expect: { requireAssertions: true },
		projects: [
			{
				extends: './vite.config.ts',
				test: {
					name: 'unit',
					environment: 'node',
					include: ['src/**/*.{test,spec}.{js,ts}'],
					exclude: ['src/**/*.svelte.{test,spec}.{js,ts}']
				}
			}
		]
	}
});
