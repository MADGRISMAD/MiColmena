<script lang="ts">
	import { resolve } from '$app/paths';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import { toast } from 'svelte-sonner';
	import { api, type ServiceCategory, type ServiceItem } from '#lib/api/index.js';
	import { catalogIcons } from '#lib/catalog-icons.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import ServiceCategoryEditor from '#lib/components/service-category-editor.svelte';
	import ServiceItemEditor from '#lib/components/service-item-editor.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { priorityLabels } from '#lib/format.js';
	import { user } from '#lib/stores/auth.js';
	import { goto } from '$app/navigation';

	let categories = $state<ServiceCategory[]>([]);
	let ticketCategories = $state<string[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	/** Qué panel está abierto: 'cat:new', 'cat:<id>', 'item:new:<catId>' o 'item:<id>'. */
	let editing = $state<string | null>(null);

	async function load() {
		error = null;
		try {
			categories = (await api.getCatalog(true)).categories;
			ticketCategories = (await api.listCategories().catch(() => ({ items: [] }))).items.map(
				(c) => c.name
			);
		} catch (err) {
			error = err instanceof Error ? err.message : 'No se pudo cargar el catálogo';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if ($user && $user.role !== 'admin') {
			goto(resolve('/(app)/tickets'), { replace: true });
			return;
		}
		if ($user) load();
	});

	async function saved() {
		editing = null;
		await load();
	}

	async function removeCategory(c: ServiceCategory) {
		if (!confirm(`¿Eliminar la categoría «${c.name}» y todos sus servicios?`)) return;
		try {
			await api.deleteServiceCategory(c.id);
			toast.success('Categoría eliminada');
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo eliminar');
		}
	}

	async function removeItem(i: ServiceItem) {
		if (!confirm(`¿Eliminar el servicio «${i.name}»?`)) return;
		try {
			await api.deleteServiceItem(i.id);
			toast.success('Servicio eliminado');
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo eliminar');
		}
	}
</script>

<PageHeader
	eyebrow="Administración"
	title="Editar catálogo"
	description="Organiza los servicios que tu equipo puede pedir."
>
	{#snippet actions()}
		<Button variant="outline" href={resolve('/(app)/catalog')}>Ver catálogo</Button>
		<Button onclick={() => (editing = 'cat:new')}>
			<PlusIcon aria-hidden="true" />
			Nueva categoría
		</Button>
	{/snippet}
</PageHeader>

{#if error}
	<Card.Root>
		<Card.Content class="flex items-center justify-between gap-4">
			<p class="text-muted-foreground">{error}</p>
			<Button variant="outline" onclick={load}>Reintentar</Button>
		</Card.Content>
	</Card.Root>
{:else if loading}
	<div class="grid gap-4" aria-busy="true">
		<Skeleton class="h-32 w-full" />
		<Skeleton class="h-32 w-full" />
	</div>
{:else}
	<div class="grid gap-4">
		{#if editing === 'cat:new'}
			<ServiceCategoryEditor onsaved={saved} oncancel={() => (editing = null)} />
		{/if}

		{#if categories.length === 0 && editing !== 'cat:new'}
			<Card.Root>
				<EmptyState
					title="El catálogo está vacío"
					description="Empieza creando una categoría y añade servicios dentro."
				>
					<Button onclick={() => (editing = 'cat:new')}>Nueva categoría</Button>
				</EmptyState>
			</Card.Root>
		{/if}

		{#each categories as category (category.id)}
			{@const Icon = catalogIcons[category.icon] ?? catalogIcons.package}
			<Card.Root>
				<Card.Content class="grid gap-4">
					{#if editing === `cat:${category.id}`}
						<ServiceCategoryEditor {category} onsaved={saved} oncancel={() => (editing = null)} />
					{:else}
						<div class="flex flex-wrap items-start justify-between gap-3">
							<div class="flex items-start gap-3">
								<span
									class="hex grid size-10 shrink-0 place-items-center bg-honey text-honey-foreground"
								>
									<Icon class="size-5" aria-hidden="true" />
								</span>
								<div>
									<h2 class="font-semibold">{category.name}</h2>
									{#if category.description}
										<p class="text-sm text-muted-foreground">{category.description}</p>
									{/if}
								</div>
							</div>
							<div class="flex gap-1">
								<Button
									variant="outline"
									size="sm"
									aria-label={`Editar categoría ${category.name}`}
									onclick={() => (editing = `cat:${category.id}`)}
								>
									<PencilIcon aria-hidden="true" />
									Editar
								</Button>
								<Button
									variant="outline"
									size="sm"
									aria-label={`Eliminar categoría ${category.name}`}
									onclick={() => removeCategory(category)}
								>
									<Trash2Icon aria-hidden="true" />
									Eliminar
								</Button>
							</div>
						</div>
					{/if}

					<ul class="grid gap-2">
						{#each category.items as item (item.id)}
							<li class="grid gap-2">
								{#if editing === `item:${item.id}`}
									<ServiceItemEditor
										{item}
										categoryId={category.id}
										{categories}
										{ticketCategories}
										onsaved={saved}
										oncancel={() => (editing = null)}
									/>
								{:else}
									<div
										class="flex flex-wrap items-center justify-between gap-2 rounded-lg border px-3 py-2"
									>
										<div class="min-w-0">
											<span class="font-medium">{item.name}</span>
											{#if !item.active}
												<Badge variant="outline" class="ml-2">Inactivo</Badge>
											{/if}
											<span class="block text-xs text-muted-foreground">
												Prioridad {priorityLabels[item.priority].toLowerCase()} · {item.fields
													.length}
												{item.fields.length === 1 ? 'campo' : 'campos'}
											</span>
										</div>
										<div class="flex gap-1">
											<Button
												variant="ghost"
												size="sm"
												aria-label={`Editar servicio ${item.name}`}
												onclick={() => (editing = `item:${item.id}`)}
											>
												<PencilIcon aria-hidden="true" />
												Editar
											</Button>
											<Button
												variant="ghost"
												size="sm"
												aria-label={`Eliminar servicio ${item.name}`}
												onclick={() => removeItem(item)}
											>
												<Trash2Icon aria-hidden="true" />
												Eliminar
											</Button>
										</div>
									</div>
								{/if}
							</li>
						{/each}
					</ul>

					{#if editing === `item:new:${category.id}`}
						<ServiceItemEditor
							categoryId={category.id}
							{categories}
							{ticketCategories}
							onsaved={saved}
							oncancel={() => (editing = null)}
						/>
					{:else}
						<div>
							<Button
								variant="outline"
								size="sm"
								aria-label={`Nuevo servicio en ${category.name}`}
								onclick={() => (editing = `item:new:${category.id}`)}
							>
								<PlusIcon aria-hidden="true" />
								Nuevo servicio
							</Button>
						</div>
					{/if}
				</Card.Content>
			</Card.Root>
		{/each}
	</div>
{/if}
