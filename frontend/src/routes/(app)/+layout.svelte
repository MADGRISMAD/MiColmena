<script lang="ts">
	import { afterNavigate, goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import ClockIcon from '@lucide/svelte/icons/clock';
	import InboxIcon from '@lucide/svelte/icons/inbox';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import LogOutIcon from '@lucide/svelte/icons/log-out';
	import MenuIcon from '@lucide/svelte/icons/menu';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import SearchIcon from '@lucide/svelte/icons/search';
	import SunIcon from '@lucide/svelte/icons/sun';
	import UserRoundCheckIcon from '@lucide/svelte/icons/user-round-check';
	import UserRoundXIcon from '@lucide/svelte/icons/user-round-x';
	import XIcon from '@lucide/svelte/icons/x';
	import { mode, toggleMode } from 'mode-watcher';
	import { api, ApiError, type Stats } from '#lib/api/index.js';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import Logo from '#lib/components/logo.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Toaster } from '#lib/components/ui/sonner/index.js';
	import { roleLabels } from '#lib/format.js';
	import { cn } from '#lib/utils.js';
	import { clearSession, isStaff, token, user } from '#lib/stores/auth.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	let loadError = $state<string | null>(null);
	let stats = $state<Stats | null>(null);
	let drawer = $state<HTMLDialogElement | null>(null);
	let searchInput = $state<HTMLInputElement | null>(null);

	// Sin sesión, a /login recordando a dónde quería ir. Con sesión, se carga el usuario si falta.
	$effect(() => {
		if (!$token) {
			const redirect = page.url.pathname + page.url.search;
			goto(resolve(`/login?redirect=${encodeURIComponent(redirect)}`), { replace: true });
			return;
		}
		if (!$user) {
			api
				.me()
				.then((me) => user.set(me))
				.catch((err) => {
					if (!(err instanceof ApiError) || err.status !== 401) {
						loadError = err instanceof Error ? err.message : 'No se pudo cargar tu sesión';
					}
				});
		}
	});

	// Los contadores de la barra lateral se refrescan en cada navegación.
	$effect(() => {
		void page.url.href;
		if (!$isStaff) return;
		api
			.stats()
			.then((s) => (stats = s))
			.catch(() => {});
	});

	afterNavigate(() => drawer?.close());

	type View = {
		href: string;
		label: string;
		icon: typeof InboxIcon;
		/** Parámetros de la URL que definen la vista; vacío = la lista sin filtros. */
		params?: Record<string, string>;
		count?: number;
	};

	const unresolved = $derived(
		stats ? stats.by_status.open + stats.by_status.in_progress + stats.by_status.waiting : undefined
	);

	const views = $derived<View[]>(
		$isStaff
			? [
					{
						href: resolve('/(app)/tickets'),
						label: 'Todos los tickets',
						icon: InboxIcon,
						params: {},
						count: unresolved
					},
					{
						href: resolve('/(app)/tickets?assignee=me'),
						label: 'Asignados a mí',
						icon: UserRoundCheckIcon,
						params: { assignee: 'me' }
					},
					{
						href: resolve('/(app)/tickets?assignee=none'),
						label: 'Sin asignar',
						icon: UserRoundXIcon,
						params: { assignee: 'none' },
						count: stats?.unassigned_open
					},
					{
						href: resolve('/(app)/tickets?status=waiting'),
						label: 'En espera del cliente',
						icon: ClockIcon,
						params: { status: 'waiting' },
						count: stats?.by_status.waiting
					}
				]
			: [{ href: resolve('/(app)/tickets'), label: 'Mis tickets', icon: InboxIcon, params: {} }]
	);

	const filterKeys = ['status', 'priority', 'assignee'];

	function isViewActive(view: View): boolean {
		const path = page.url.pathname;
		const list = resolve('/(app)/tickets');
		// El detalle de un ticket cuenta como "Todos los tickets".
		if (path.startsWith(list + '/') && path !== resolve('/(app)/tickets/new')) {
			return Object.keys(view.params ?? {}).length === 0;
		}
		if (path !== list) return false;
		const params = page.url.searchParams;
		return filterKeys.every((key) => (params.get(key) ?? '') === (view.params?.[key] ?? ''));
	}

	// Búsqueda global, como en las herramientas de soporte: "/" la enfoca desde cualquier página.
	let search = $derived(
		page.url.pathname === resolve('/(app)/tickets') ? (page.url.searchParams.get('q') ?? '') : ''
	);

	function submitSearch(event: SubmitEvent) {
		event.preventDefault();
		const q = search.trim();
		goto(q ? resolve(`/(app)/tickets?q=${encodeURIComponent(q)}`) : resolve('/(app)/tickets'));
	}

	function onKeydown(event: KeyboardEvent) {
		const target = event.target as HTMLElement | null;
		const typing = target?.closest('input, textarea, select, [contenteditable="true"]');
		if (event.key === '/' && !typing && !event.metaKey && !event.ctrlKey) {
			event.preventDefault();
			searchInput?.focus();
		}
	}

	function logout() {
		clearSession();
	}
</script>

<svelte:window onkeydown={onKeydown} />

<Toaster richColors position="top-right" />

{#snippet sidebar()}
	<div class="flex h-16 shrink-0 items-center px-5">
		<a href={resolve('/')} aria-label="MiColmena, inicio"><Logo tone="dark" /></a>
	</div>

	<nav class="flex-1 space-y-6 overflow-y-auto px-3 py-2" aria-label="Navegación principal">
		{#if $isStaff}
			<div>
				<p
					class="px-3 pb-2 text-[0.68rem] font-semibold tracking-wider text-sidebar-foreground/50 uppercase"
				>
					Resumen
				</p>
				<a
					href={resolve('/(app)/dashboard')}
					aria-current={page.url.pathname === resolve('/(app)/dashboard') ? 'page' : undefined}
					class={cn(
						'nav-item',
						page.url.pathname === resolve('/(app)/dashboard') && 'nav-item-active'
					)}
				>
					<LayoutDashboardIcon class="size-4" aria-hidden="true" />
					Dashboard
				</a>
			</div>
		{/if}

		<div>
			<p
				class="px-3 pb-2 text-[0.68rem] font-semibold tracking-wider text-sidebar-foreground/50 uppercase"
			>
				{$isStaff ? 'Vistas' : 'Soporte'}
			</p>
			<ul class="space-y-0.5">
				{#each views as view (view.href)}
					{@const active = isViewActive(view)}
					<li>
						<a
							href={view.href}
							aria-current={active ? 'page' : undefined}
							class={cn('nav-item', active && 'nav-item-active')}
						>
							<view.icon class="size-4" aria-hidden="true" />
							<span class="flex-1 truncate">{view.label}</span>
							{#if view.count !== undefined && view.count > 0}
								<span
									class="rounded-full bg-white/10 px-2 py-0.5 text-[0.7rem] font-medium tabular-nums"
								>
									{view.count}
								</span>
							{/if}
						</a>
					</li>
				{/each}
			</ul>
		</div>
	</nav>

	{#if $user}
		<div class="flex items-center gap-3 border-t border-sidebar-border p-4">
			<HexAvatar name={$user.name} id={$user.id} size="md" />
			<div class="min-w-0 flex-1">
				<p class="truncate text-sm font-medium text-white">{$user.name}</p>
				<p class="text-xs text-sidebar-foreground/60">{roleLabels[$user.role]}</p>
			</div>
			<button
				type="button"
				class="rounded-md p-2 text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-white"
				onclick={toggleMode}
				aria-label={mode.current === 'dark' ? 'Usar tema claro' : 'Usar tema oscuro'}
				title={mode.current === 'dark' ? 'Tema claro' : 'Tema oscuro'}
			>
				{#if mode.current === 'dark'}
					<SunIcon class="size-4" aria-hidden="true" />
				{:else}
					<MoonIcon class="size-4" aria-hidden="true" />
				{/if}
			</button>
			<button
				type="button"
				class="rounded-md p-2 text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-white"
				onclick={logout}
				aria-label="Cerrar sesión"
				title="Cerrar sesión"
			>
				<LogOutIcon class="size-4" aria-hidden="true" />
			</button>
		</div>
	{/if}
{/snippet}

{#if loadError}
	<div class="flex min-h-screen flex-col items-center justify-center gap-4 p-4 text-center">
		<p class="text-muted-foreground">{loadError}</p>
		<Button onclick={() => location.reload()}>Reintentar</Button>
	</div>
{:else if !$user}
	<div class="flex min-h-screen items-center justify-center" aria-busy="true">
		<Skeleton class="h-8 w-48" />
	</div>
{:else}
	<div class="min-h-screen lg:grid lg:grid-cols-[16rem_minmax(0,1fr)] lg:bg-sidebar">
		<aside
			class="bg-honeycomb-dark sticky top-0 hidden h-screen flex-col bg-sidebar text-sidebar-foreground lg:flex"
		>
			{@render sidebar()}
		</aside>

		<!-- En pantallas pequeñas, la misma barra lateral como panel deslizable. -->
		<dialog
			bind:this={drawer}
			class="bg-honeycomb-dark m-0 h-full max-h-none w-72 max-w-[85vw] flex-col bg-sidebar p-0 text-sidebar-foreground backdrop:bg-black/50 open:flex lg:hidden"
			aria-label="Menú"
			onclick={(e) => e.target === drawer && drawer?.close()}
		>
			<button
				type="button"
				class="absolute top-4 right-3 rounded-md p-2 text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-white"
				onclick={() => drawer?.close()}
				aria-label="Cerrar menú"
			>
				<XIcon class="size-4" aria-hidden="true" />
			</button>
			{@render sidebar()}
		</dialog>

		<div class="flex min-h-screen min-w-0 flex-col bg-background">
			<header
				class="sticky top-0 z-20 flex h-16 items-center gap-3 border-b bg-background/85 px-4 backdrop-blur md:px-8"
			>
				<button
					type="button"
					class="-ml-2 rounded-md p-2 text-muted-foreground hover:bg-accent lg:hidden"
					onclick={() => drawer?.showModal()}
					aria-label="Abrir menú"
				>
					<MenuIcon class="size-5" aria-hidden="true" />
				</button>

				<form role="search" class="relative max-w-xl flex-1" onsubmit={submitSearch}>
					<SearchIcon
						class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
						aria-hidden="true"
					/>
					<input
						bind:this={searchInput}
						bind:value={search}
						type="search"
						aria-label="Buscar tickets"
						placeholder="Buscar tickets…"
						class="h-9 w-full rounded-md border border-input bg-card pr-10 pl-9 text-sm shadow-xs outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
					/>
					<kbd
						class="pointer-events-none absolute top-1/2 right-2.5 hidden -translate-y-1/2 rounded border bg-muted px-1.5 font-mono text-[0.7rem] text-muted-foreground sm:block"
						>/</kbd
					>
				</form>

				<Button href={resolve('/(app)/tickets/new')} class="ml-auto shrink-0">
					<PlusIcon aria-hidden="true" />
					<span class="hidden sm:inline">Nuevo ticket</span>
					<span class="sr-only sm:hidden">Nuevo ticket</span>
				</Button>
			</header>

			<main class="flex-1 px-4 py-6 md:px-8 md:py-8">
				<div class="mx-auto max-w-7xl">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{/if}

<style>
	:global(.nav-item) {
		position: relative;
		display: flex;
		align-items: center;
		gap: 0.625rem;
		border-radius: 0.375rem;
		padding: 0.5rem 0.75rem;
		font-size: 0.875rem;
		color: var(--sidebar-foreground);
		transition:
			background-color 120ms,
			color 120ms;
	}
	:global(.nav-item:hover) {
		background: var(--sidebar-accent);
		color: var(--sidebar-accent-foreground);
	}
	:global(.nav-item-active) {
		background: var(--sidebar-accent);
		color: white;
		font-weight: 500;
	}
	/* Una barrita de miel marca la vista actual. */
	:global(.nav-item-active)::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0.4rem;
		bottom: 0.4rem;
		width: 3px;
		border-radius: 999px;
		background: var(--honey);
	}
</style>
