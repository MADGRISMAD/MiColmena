<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import HistoryIcon from '@lucide/svelte/icons/history';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import { toast } from 'svelte-sonner';
	import { api, ApiError, type AssetDetail, type AssetInput } from '#lib/api/index.js';
	import AssetForm from '#lib/components/asset-form.svelte';
	import AssetStateBadge from '#lib/components/asset-state-badge.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import {
		assetCategoryLabels,
		describeAssetEvent,
		formatDateTime,
		formatRelative
	} from '#lib/format.js';
	import { isStaff, user } from '#lib/stores/auth.js';
	import { cn } from '#lib/utils.js';

	const assetId = $derived(Number(page.params.id));

	type Tab = 'general' | 'activity' | 'tickets';
	let tab = $state<Tab>('general');
	let detail = $state<AssetDetail | null>(null);
	let error = $state<string | null>(null);
	let deleting = $state(false);

	async function load(id: number, quiet = false) {
		if (!quiet) {
			error = null;
			detail = null;
		}
		try {
			detail = await api.getAsset(id);
		} catch (err) {
			error =
				err instanceof ApiError && err.status === 404
					? 'Este equipo no existe.'
					: err instanceof Error
						? err.message
						: 'No se pudo cargar el equipo';
		}
	}

	$effect(() => {
		if (!$isStaff) {
			goto(resolve('/(app)/tickets'), { replace: true });
			return;
		}
		load(assetId);
	});

	async function save(input: AssetInput) {
		await api.updateAsset(assetId, input);
		await load(assetId, true);
		toast.success('Equipo actualizado');
	}

	async function remove() {
		if (!detail) return;
		if (!confirm(`¿Eliminar el equipo ${detail.asset.tag}? Esta acción no se puede deshacer.`)) {
			return;
		}
		deleting = true;
		try {
			await api.deleteAsset(assetId);
			toast.success('Equipo eliminado');
			await goto(resolve('/(app)/assets'));
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo eliminar');
			deleting = false;
		}
	}

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'general', label: 'General' },
		{ id: 'activity', label: 'Actividad' },
		{ id: 'tickets', label: 'Tickets' }
	];

	// Los eventos de ticket traen "#N": se enlazan al ticket.
	const ticketNumber = (value: string) => /^#(\d+)$/.exec(value)?.[1] ?? null;
</script>

<svelte:head>
	<title>{detail ? `${detail.asset.tag} ${detail.asset.name}` : 'Equipo'} · MiColmena</title>
</svelte:head>

{#if error}
	<Card.Root>
		<EmptyState title={error}>
			<Button variant="outline" href={resolve('/(app)/assets')}>Ver activos</Button>
		</EmptyState>
	</Card.Root>
{:else if !detail}
	<div class="grid gap-4" aria-busy="true">
		<Skeleton class="h-4 w-40" />
		<Skeleton class="h-9 w-2/3" />
		<Skeleton class="h-72" />
	</div>
{:else}
	{@const asset = detail.asset}
	<nav class="mb-3 flex items-center gap-1 text-sm text-muted-foreground" aria-label="Ruta">
		<a href={resolve('/(app)/assets')} class="hover:text-foreground">Activos</a>
		<ChevronRightIcon class="size-3.5" aria-hidden="true" />
		<span class="font-mono">{asset.tag}</span>
	</nav>

	<div class="mb-6">
		<h1 class="text-2xl font-bold tracking-tight text-balance">
			<span class="font-mono text-muted-foreground">{asset.tag}</span>
			{asset.name}
		</h1>
		<div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-muted-foreground">
			<AssetStateBadge state={asset.state} />
			<span class="rounded-md bg-muted px-2 py-0.5 text-xs font-medium text-foreground">
				{assetCategoryLabels[asset.category]}
			</span>
			<span>
				Responsable:
				<span class="text-foreground">{asset.assigned_to?.name ?? 'Sin asignar'}</span>
			</span>
			{#if asset.location}<span>Ubicación: {asset.location}</span>{/if}
		</div>
	</div>

	<div class="mb-4 flex gap-1 border-b" role="tablist" aria-label="Secciones del equipo">
		{#each tabs as t (t.id)}
			<button
				type="button"
				role="tab"
				id={`tab-${t.id}`}
				aria-selected={tab === t.id}
				aria-controls={`panel-${t.id}`}
				onclick={() => (tab = t.id)}
				class={cn(
					'-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors',
					tab === t.id
						? 'border-primary text-foreground'
						: 'border-transparent text-muted-foreground hover:text-foreground'
				)}
			>
				{t.label}
				{#if t.id === 'tickets'}
					<span class="ml-1 text-xs text-muted-foreground">({detail.tickets.length})</span>
				{/if}
			</button>
		{/each}
	</div>

	{#if tab === 'general'}
		<div role="tabpanel" id="panel-general" aria-labelledby="tab-general">
			<Card.Root class="max-w-3xl">
				<Card.Content>
					<!-- Se vuelve a crear tras guardar para mostrar lo que decidió el servidor (p. ej. el estado). -->
					{#key asset.updated_at}
						<AssetForm {asset} submitLabel="Guardar cambios" onsubmit={save}>
							{#snippet actions()}
								{#if $user?.role === 'admin'}
									<Button
										type="button"
										variant="outline"
										class="ml-auto text-destructive"
										disabled={deleting}
										onclick={remove}
									>
										<Trash2Icon aria-hidden="true" />
										Eliminar equipo
									</Button>
								{/if}
							{/snippet}
						</AssetForm>
					{/key}
				</Card.Content>
			</Card.Root>
		</div>
	{:else if tab === 'activity'}
		<div role="tabpanel" id="panel-activity" aria-labelledby="tab-activity">
			<Card.Root class="max-w-3xl">
				<Card.Content>
					{#if detail.events.length === 0}
						<EmptyState title="Sin actividad" description="Aún no hay cambios registrados." />
					{:else}
						<ol class="grid gap-4">
							{#each detail.events as e (e.id)}
								{@const n = e.kind === 'ticket' ? ticketNumber(e.new_value) : null}
								<li class="flex items-start gap-3 text-sm">
									<span class="hex mt-0.5 grid size-6 shrink-0 place-items-center bg-muted">
										<HistoryIcon class="size-3" aria-hidden="true" />
									</span>
									<div>
										<p class="font-medium">
											{#if n}
												Vinculado al ticket
												<a
													href={resolve('/(app)/tickets/[id]', { id: n })}
													class="text-primary underline-offset-4 hover:underline dark:text-honey"
													>#{n}</a
												>
											{:else}
												{describeAssetEvent(e)}
											{/if}
										</p>
										<p class="text-xs text-muted-foreground">
											{e.actor?.name ?? 'Sistema'} ·
											<time datetime={e.created_at} title={formatDateTime(e.created_at)}>
												{formatRelative(e.created_at)}
											</time>
										</p>
									</div>
								</li>
							{/each}
						</ol>
					{/if}
				</Card.Content>
			</Card.Root>
		</div>
	{:else}
		<div role="tabpanel" id="panel-tickets" aria-labelledby="tab-tickets">
			<Card.Root class="max-w-3xl">
				<Card.Content>
					{#if detail.tickets.length === 0}
						<EmptyState
							title="Sin tickets"
							description="Cuando un ticket se vincule a este equipo aparecerá aquí."
						/>
					{:else}
						<ul class="divide-y">
							{#each detail.tickets as t (t.id)}
								<li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-3 text-sm">
									<span class="font-mono text-muted-foreground">#{t.id}</span>
									<a
										href={resolve('/(app)/tickets/[id]', { id: String(t.id) })}
										class="min-w-0 flex-1 font-medium hover:underline"
									>
										{t.title}
									</a>
									<StatusBadge status={t.status} />
									<PriorityBadge priority={t.priority} />
								</li>
							{/each}
						</ul>
						<div class="mt-4">
							<a
								href={resolve(`/(app)/tickets?asset=${asset.id}`)}
								class="text-sm font-medium text-primary underline-offset-4 hover:underline dark:text-honey"
							>
								Ver en la lista de tickets
							</a>
						</div>
					{/if}
				</Card.Content>
			</Card.Root>
		</div>
	{/if}
{/if}
