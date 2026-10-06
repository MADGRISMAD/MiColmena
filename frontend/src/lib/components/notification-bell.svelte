<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import AtSignIcon from '@lucide/svelte/icons/at-sign';
	import BellIcon from '@lucide/svelte/icons/bell';
	import CheckCheckIcon from '@lucide/svelte/icons/check-check';
	import CircleDotIcon from '@lucide/svelte/icons/circle-dot';
	import MessageSquareIcon from '@lucide/svelte/icons/message-square';
	import TicketPlusIcon from '@lucide/svelte/icons/ticket-plus';
	import UserRoundCheckIcon from '@lucide/svelte/icons/user-round-check';
	import { api, type Notification, type NotificationKind } from '#lib/api/index.js';
	import { formatRelative } from '#lib/format.js';
	import { liveEvent, notificationsVersion } from '#lib/live/index.js';
	import { cn } from '#lib/utils.js';

	let items = $state<Notification[]>([]);
	let unread = $state(0);
	let open = $state(false);
	let root = $state<HTMLDivElement | null>(null);

	async function load() {
		try {
			const res = await api.listNotifications(20);
			items = res.items;
			unread = res.unread;
		} catch {
			// La campana no debe romper la página si falla.
		}
	}

	// Se recarga al abrir la app, con cada aviso en tiempo real y al cambiar de página.
	$effect(() => {
		void $liveEvent.seq;
		void $notificationsVersion;
		void page.url.pathname;
		load();
	});

	const icons: Record<NotificationKind, typeof BellIcon> = {
		new_ticket: TicketPlusIcon,
		assigned: UserRoundCheckIcon,
		comment: MessageSquareIcon,
		mention: AtSignIcon,
		status: CircleDotIcon
	};

	async function openNotification(n: Notification) {
		open = false;
		if (!n.read_at) {
			unread = Math.max(0, unread - 1);
			n.read_at = new Date().toISOString();
			api.readNotifications({ ids: [n.id] }).catch(() => {});
		}
		if (n.ticket_id) await goto(resolve('/(app)/tickets/[id]', { id: String(n.ticket_id) }));
	}

	async function readAll() {
		unread = 0;
		items = items.map((n) => ({ ...n, read_at: n.read_at ?? new Date().toISOString() }));
		await api.readNotifications().catch(() => {});
	}

	function onWindowClick(event: MouseEvent) {
		if (open && root && !root.contains(event.target as Node)) open = false;
	}
</script>

<svelte:window
	onclick={onWindowClick}
	onkeydown={(e) => {
		if (e.key === 'Escape') open = false;
	}}
/>

<div class="relative" bind:this={root}>
	<button
		type="button"
		class="relative rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-foreground"
		aria-label={unread > 0 ? `Notificaciones (${unread} sin leer)` : 'Notificaciones'}
		aria-expanded={open}
		aria-haspopup="true"
		onclick={() => (open = !open)}
	>
		<BellIcon class="size-5" aria-hidden="true" />
		{#if unread > 0}
			<span
				class="absolute top-1 right-1 grid min-w-4 place-items-center rounded-full bg-honey px-1 text-[0.62rem] leading-4 font-bold text-honey-foreground tabular-nums"
				aria-hidden="true"
			>
				{unread > 99 ? '99+' : unread}
			</span>
		{/if}
	</button>

	{#if open}
		<div
			class="absolute right-0 z-40 mt-2 w-[min(24rem,calc(100vw-2rem))] overflow-hidden rounded-xl border bg-popover text-popover-foreground shadow-xl"
			role="region"
			aria-label="Notificaciones"
		>
			<div class="flex items-center justify-between border-b px-4 py-3">
				<p class="text-sm font-semibold">Notificaciones</p>
				{#if unread > 0}
					<button
						type="button"
						class="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline dark:text-honey"
						onclick={readAll}
					>
						<CheckCheckIcon class="size-3.5" aria-hidden="true" />
						Marcar todas como leídas
					</button>
				{/if}
			</div>
			{#if items.length === 0}
				<p class="px-4 py-10 text-center text-sm text-muted-foreground">
					No tienes notificaciones todavía.
				</p>
			{:else}
				<ul class="max-h-[60vh] divide-y overflow-y-auto">
					{#each items as n (n.id)}
						{@const Icon = icons[n.kind] ?? BellIcon}
						<li>
							<button
								type="button"
								class={cn(
									'flex w-full gap-3 px-4 py-3 text-left text-sm hover:bg-muted/60',
									!n.read_at && 'bg-honey/10'
								)}
								onclick={() => openNotification(n)}
							>
								<span
									class="hex grid size-8 shrink-0 place-items-center bg-muted text-muted-foreground"
								>
									<Icon class="size-4" aria-hidden="true" />
								</span>
								<span class="min-w-0 flex-1">
									<span class="block font-medium">{n.summary}</span>
									{#if n.ticket_title}
										<span class="block truncate text-muted-foreground">
											#{n.ticket_id} · {n.ticket_title}
										</span>
									{/if}
									<span class="text-xs text-muted-foreground">{formatRelative(n.created_at)}</span>
								</span>
								{#if !n.read_at}
									<span class="mt-1.5 size-2 shrink-0 rounded-full bg-honey" aria-label="Sin leer"
									></span>
								{/if}
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{/if}
</div>
