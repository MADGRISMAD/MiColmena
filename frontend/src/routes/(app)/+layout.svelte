<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import LogOutIcon from '@lucide/svelte/icons/log-out';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import TicketIcon from '@lucide/svelte/icons/ticket';
	import { api, ApiError } from '#lib/api/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Toaster } from '#lib/components/ui/sonner/index.js';
	import { roleLabels } from '#lib/format.js';
	import { cn } from '#lib/utils.js';
	import { clearSession, isStaff, token, user } from '#lib/stores/auth.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	let loadError = $state<string | null>(null);

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

	const nav = $derived([
		...($isStaff
			? [{ href: resolve('/(app)/dashboard'), label: 'Dashboard', icon: LayoutDashboardIcon }]
			: []),
		{ href: resolve('/(app)/tickets'), label: 'Tickets', icon: TicketIcon },
		{ href: resolve('/(app)/tickets/new'), label: 'Nuevo ticket', icon: PlusIcon }
	]);

	function isActive(href: string) {
		const path = page.url.pathname;
		if (href === resolve('/(app)/tickets')) {
			return (
				path === href || (path.startsWith(href + '/') && path !== resolve('/(app)/tickets/new'))
			);
		}
		return path === href;
	}

	function logout() {
		clearSession();
	}
</script>

<Toaster richColors position="top-right" />

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
	<div class="flex min-h-screen flex-col md:flex-row">
		<aside class="flex shrink-0 flex-col border-b bg-sidebar md:w-60 md:border-r md:border-b-0">
			<a href={resolve('/')} class="flex h-14 items-center gap-2 px-4 font-semibold">
				<img src="/logo.png" alt="" class="size-7" />
				MiColmena
			</a>
			<nav class="flex gap-1 overflow-x-auto px-2 pb-2 md:flex-1 md:flex-col md:pb-0">
				{#each nav as item (item.href)}
					<a
						href={item.href}
						aria-current={isActive(item.href) ? 'page' : undefined}
						class={cn(
							'flex items-center gap-2 rounded-md px-3 py-2 text-sm whitespace-nowrap text-sidebar-foreground hover:bg-sidebar-accent',
							isActive(item.href) && 'bg-sidebar-accent font-medium'
						)}
					>
						<item.icon class="size-4" aria-hidden="true" />
						{item.label}
					</a>
				{/each}
			</nav>
			<div class="hidden border-t p-4 md:block">
				<p class="truncate text-sm font-medium">{$user.name}</p>
				<p class="text-xs text-muted-foreground">{roleLabels[$user.role]}</p>
				<Button variant="outline" size="sm" class="mt-3 w-full" onclick={logout}>
					<LogOutIcon aria-hidden="true" />
					Cerrar sesión
				</Button>
			</div>
		</aside>

		<main class="min-w-0 flex-1">
			<div class="flex justify-end border-b px-4 py-2 md:hidden">
				<Button variant="ghost" size="sm" onclick={logout}>
					<LogOutIcon aria-hidden="true" />
					Cerrar sesión
				</Button>
			</div>
			<div class="mx-auto max-w-6xl p-4 md:p-8">
				{@render children()}
			</div>
		</main>
	</div>
{/if}
