<script lang="ts">
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import { toast } from 'svelte-sonner';
	import { api, ApiError, type Category, type SlaPolicy } from '#lib/api/index.js';
	import PageHeader from '#lib/components/page-header.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatMinutes } from '#lib/sla.js';

	// --- Categorías ---
	let categories = $state<Category[] | null>(null);
	let newCategory = $state('');
	let categoryError = $state('');

	async function loadCategories() {
		categories = await api
			.listCategories()
			.then((r) => r.items)
			.catch(() => []);
	}

	async function addCategory(event: SubmitEvent) {
		event.preventDefault();
		categoryError = '';
		try {
			await api.createCategory(newCategory.trim());
			newCategory = '';
			await loadCategories();
		} catch (err) {
			categoryError =
				err instanceof ApiError ? (err.fields.name ?? err.message) : 'No se pudo crear';
		}
	}

	async function rename(c: Category) {
		const name = prompt('Nuevo nombre (se actualiza también en los tickets)', c.name)?.trim();
		if (!name || name === c.name) return;
		try {
			await api.renameCategory(c.id, name);
			await loadCategories();
			toast.success('Categoría renombrada');
		} catch (err) {
			toast.error(
				err instanceof ApiError ? (err.fields.name ?? err.message) : 'No se pudo renombrar'
			);
		}
	}

	async function remove(c: Category) {
		if (!confirm(`¿Quitar «${c.name}» de la lista? Los tickets que la usan la conservan.`)) return;
		await api.deleteCategory(c.id).catch(() => {});
		await loadCategories();
	}

	// --- SLA, editado en horas ---
	let sla = $state<{ priority: SlaPolicy['priority']; first: number; resolution: number }[] | null>(
		null
	);
	let savingSla = $state(false);

	async function loadSla() {
		const items = await api
			.listSla()
			.then((r) => r.items)
			.catch(() => []);
		sla = items.map((p) => ({
			priority: p.priority,
			first: p.first_response_minutes / 60,
			resolution: p.resolution_minutes / 60
		}));
	}

	async function saveSla(event: SubmitEvent) {
		event.preventDefault();
		if (!sla) return;
		savingSla = true;
		try {
			await api.updateSla(
				sla.map((p) => ({
					priority: p.priority,
					first_response_minutes: Math.round(p.first * 60),
					resolution_minutes: Math.round(p.resolution * 60)
				}))
			);
			toast.success('Plazos guardados');
			await loadSla();
		} catch (err) {
			const fields = err instanceof ApiError ? Object.values(err.fields) : [];
			toast.error(fields[0] ?? (err instanceof Error ? err.message : 'No se pudo guardar'));
		} finally {
			savingSla = false;
		}
	}

	$effect(() => {
		loadCategories();
		loadSla();
	});
</script>

<PageHeader
	eyebrow="Administración"
	title="Configuración"
	description="Categorías de los tickets y plazos de atención."
/>

<div class="grid gap-6 xl:grid-cols-2">
	<Card.Root>
		<Card.Header>
			<Card.Title>Categorías</Card.Title>
			<Card.Description>Los clientes eligen una al abrir un ticket.</Card.Description>
		</Card.Header>
		<Card.Content class="grid gap-4">
			<form class="flex gap-2" onsubmit={addCategory}>
				<Input
					bind:value={newCategory}
					placeholder="Nueva categoría"
					aria-label="Nueva categoría"
					aria-invalid={!!categoryError}
				/>
				<Button type="submit" disabled={!newCategory.trim()}>Añadir</Button>
			</form>
			{#if categoryError}<p class="text-sm text-destructive">{categoryError}</p>{/if}
			{#if categories === null}
				<Skeleton class="h-24" />
			{:else if categories.length === 0}
				<p class="text-sm text-muted-foreground">Aún no hay categorías.</p>
			{:else}
				<ul class="divide-y rounded-lg border">
					{#each categories as c (c.id)}
						<li class="flex items-center gap-2 px-3 py-2 text-sm">
							<span class="flex-1">{c.name}</span>
							<Button
								variant="ghost"
								size="icon-sm"
								onclick={() => rename(c)}
								aria-label={`Renombrar ${c.name}`}
							>
								<PencilIcon aria-hidden="true" />
							</Button>
							<Button
								variant="ghost"
								size="icon-sm"
								onclick={() => remove(c)}
								aria-label={`Quitar ${c.name}`}
							>
								<Trash2Icon aria-hidden="true" />
							</Button>
						</li>
					{/each}
				</ul>
			{/if}
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header>
			<Card.Title>Plazos de atención (SLA)</Card.Title>
			<Card.Description>
				Horas desde que se abre el ticket. Se cuentan las 24 horas del día.
			</Card.Description>
		</Card.Header>
		<Card.Content>
			{#if sla === null}
				<Skeleton class="h-40" />
			{:else}
				<form onsubmit={saveSla} class="grid gap-4">
					<table class="w-full text-sm">
						<thead class="text-left text-xs text-muted-foreground">
							<tr>
								<th class="pb-2 font-medium">Prioridad</th>
								<th class="pb-2 font-medium">Primera respuesta (h)</th>
								<th class="pb-2 font-medium">Resolución (h)</th>
							</tr>
						</thead>
						<tbody>
							{#each sla as row (row.priority)}
								<tr>
									<td class="py-1.5 pr-3"><PriorityBadge priority={row.priority} /></td>
									<td class="py-1.5 pr-3">
										<Input
											type="number"
											min="0.25"
											step="0.25"
											bind:value={row.first}
											aria-label={`Primera respuesta, prioridad ${row.priority}`}
											title={formatMinutes(row.first * 60)}
										/>
									</td>
									<td class="py-1.5">
										<Input
											type="number"
											min="0.25"
											step="0.25"
											bind:value={row.resolution}
											aria-label={`Resolución, prioridad ${row.priority}`}
											title={formatMinutes(row.resolution * 60)}
										/>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
					<div>
						<Button type="submit" disabled={savingSla}>
							{savingSla ? 'Guardando…' : 'Guardar plazos'}
						</Button>
					</div>
				</form>
			{/if}
		</Card.Content>
	</Card.Root>
</div>
