<script lang="ts">
	import { resolve } from '$app/paths';
	import { api } from '#lib/api/index.js';
	import Logo from '#lib/components/logo.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Toaster } from '#lib/components/ui/sonner/index.js';
	import { isAuthenticated, isStaff, token, user } from '#lib/stores/auth.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	// La ayuda es pública; si hay sesión se carga el usuario para mostrar las opciones del equipo.
	$effect(() => {
		if ($token && !$user)
			api
				.me()
				.then((me) => user.set(me))
				.catch(() => {});
	});
</script>

<Toaster richColors position="top-right" />

<div class="flex min-h-screen flex-col">
	<header class="sticky top-0 z-30 border-b bg-background/85 backdrop-blur">
		<div class="mx-auto flex h-16 max-w-5xl items-center justify-between gap-4 px-4">
			<a href={resolve('/help')} class="flex items-center gap-3" aria-label="Centro de ayuda">
				<Logo />
				<span class="hidden border-l pl-3 text-sm text-muted-foreground sm:inline"
					>Centro de ayuda</span
				>
			</a>
			<div class="flex items-center gap-2">
				{#if $isStaff}
					<Button variant="outline" size="sm" href={resolve('/help/new')}>Nuevo artículo</Button>
				{/if}
				{#if $isAuthenticated}
					<Button size="sm" href={resolve('/(app)/tickets')}>Ir a mis tickets</Button>
				{:else}
					<Button variant="ghost" size="sm" href={resolve('/login')}>Iniciar sesión</Button>
				{/if}
			</div>
		</div>
	</header>
	<main class="mx-auto w-full max-w-5xl flex-1 px-4 py-10">
		{@render children()}
	</main>
</div>
