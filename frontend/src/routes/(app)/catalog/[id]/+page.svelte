<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { toast } from 'svelte-sonner';
	import { api, ApiError, type Asset, type ServiceItem } from '#lib/api/index.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import FormField from '#lib/components/form-field.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';

	let item = $state<ServiceItem | null>(null);
	let assets = $state<Asset[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let values = $state<Record<string, string>>({});
	let notes = $state('');
	let assetId = $state('');
	let errors = $state<Record<string, string>>({});
	let formError = $state<string | null>(null);
	let submitting = $state(false);

	const itemId = $derived(Number(page.params.id));

	async function load(id: number) {
		loading = true;
		error = null;
		try {
			const res = await api.getCatalog();
			item = res.categories.flatMap((c) => c.items).find((i) => i.id === id) ?? null;
			values = {};
			errors = {};
			if (item) assets = (await api.myAssets().catch(() => ({ items: [] as Asset[] }))).items;
		} catch (err) {
			error = err instanceof Error ? err.message : 'No se pudo cargar el servicio';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		load(itemId);
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!item || submitting) return;
		formError = null;
		const next: Record<string, string> = {};
		for (const f of item.fields) {
			if (f.required && !values[f.key]?.trim()) next[f.key] = 'Este campo es obligatorio';
		}
		errors = next;
		if (Object.keys(next).length > 0) return;

		const payload: Record<string, string | number> = {};
		for (const f of item.fields) {
			const v = values[f.key]?.trim();
			if (!v) continue;
			payload[f.key] = f.type === 'number' && !Number.isNaN(Number(v)) ? Number(v) : v;
		}

		submitting = true;
		try {
			const ticket = await api.requestService(item.id, {
				values: payload,
				notes: notes.trim() || undefined,
				asset_id: assetId ? Number(assetId) : undefined
			});
			toast.success(`Solicitud #${ticket.id} creada`);
			await goto(resolve('/(app)/tickets/[id]', { id: String(ticket.id) }));
		} catch (err) {
			if (err instanceof ApiError && Object.keys(err.fields).length > 0) {
				const mapped: Record<string, string> = {};
				for (const [key, msg] of Object.entries(err.fields)) {
					mapped[key.replace(/^values\./, '')] = msg.charAt(0).toUpperCase() + msg.slice(1);
				}
				errors = mapped;
			} else {
				formError = err instanceof Error ? err.message : 'No se pudo crear la solicitud';
			}
		} finally {
			submitting = false;
		}
	}
</script>

{#if loading}
	<div class="grid gap-4" aria-busy="true">
		<Skeleton class="h-8 w-64" />
		<Skeleton class="h-64 w-full" />
	</div>
{:else if error}
	<Card.Root>
		<Card.Content class="flex items-center justify-between gap-4">
			<p class="text-muted-foreground">{error}</p>
			<Button variant="outline" onclick={() => load(itemId)}>Reintentar</Button>
		</Card.Content>
	</Card.Root>
{:else if !item}
	<Card.Root>
		<EmptyState title="Este servicio no existe" description="Puede que ya no esté disponible.">
			<Button href={resolve('/(app)/catalog')}>Volver al catálogo</Button>
		</EmptyState>
	</Card.Root>
{:else}
	<PageHeader eyebrow="Catálogo de servicios" title={item.name} description={item.description} />

	<Card.Root class="max-w-2xl">
		<Card.Content>
			<form onsubmit={submit} class="grid gap-5" novalidate>
				{#if formError}
					<Alert.Root variant="destructive">
						<Alert.Description>{formError}</Alert.Description>
					</Alert.Root>
				{/if}

				{#each item.fields as field (field.key)}
					<FormField
						id={`f-${field.key}`}
						label={field.required ? `${field.label} *` : field.label}
						errors={errors[field.key] ? [errors[field.key]] : undefined}
					>
						{#snippet children({ id, invalid, describedBy })}
							{#if field.type === 'textarea'}
								<Textarea
									{id}
									rows={4}
									bind:value={values[field.key]}
									aria-invalid={invalid}
									aria-describedby={describedBy}
									aria-required={field.required}
								/>
							{:else if field.type === 'select'}
								<NativeSelect
									{id}
									class="w-full"
									bind:value={values[field.key]}
									aria-invalid={invalid}
									aria-describedby={describedBy}
									aria-required={field.required}
								>
									<NativeSelectOption value="">Selecciona una opción</NativeSelectOption>
									{#each field.options ?? [] as option (option)}
										<NativeSelectOption value={option}>{option}</NativeSelectOption>
									{/each}
								</NativeSelect>
							{:else}
								<Input
									{id}
									type={field.type === 'number'
										? 'number'
										: field.type === 'date'
											? 'date'
											: 'text'}
									bind:value={values[field.key]}
									aria-invalid={invalid}
									aria-describedby={describedBy}
									aria-required={field.required}
								/>
							{/if}
						{/snippet}
					</FormField>
				{/each}

				{#if assets.length > 0}
					<FormField
						id="asset"
						label="¿Sobre qué equipo? (opcional)"
						errors={errors.asset_id ? [errors.asset_id] : undefined}
					>
						{#snippet children({ id, invalid, describedBy })}
							<NativeSelect
								{id}
								class="w-full"
								bind:value={assetId}
								aria-invalid={invalid}
								aria-describedby={describedBy}
							>
								<NativeSelectOption value="">Ninguno</NativeSelectOption>
								{#each assets as asset (asset.id)}
									<NativeSelectOption value={String(asset.id)}>
										{asset.name} ({asset.tag})
									</NativeSelectOption>
								{/each}
							</NativeSelect>
						{/snippet}
					</FormField>
				{/if}

				<FormField
					id="notes"
					label="Notas adicionales (opcional)"
					errors={errors.notes ? [errors.notes] : undefined}
				>
					{#snippet children({ id, invalid, describedBy })}
						<Textarea
							{id}
							rows={3}
							bind:value={notes}
							aria-invalid={invalid}
							aria-describedby={describedBy}
						/>
					{/snippet}
				</FormField>

				<div class="flex gap-2">
					<Button type="submit" disabled={submitting}>
						{submitting ? 'Enviando…' : 'Enviar solicitud'}
					</Button>
					<Button variant="ghost" href={resolve('/(app)/catalog')}>Cancelar</Button>
				</div>
			</form>
		</Card.Content>
	</Card.Root>
{/if}
