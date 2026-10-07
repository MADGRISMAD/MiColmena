<script lang="ts">
	import { api, ApiError, ASSET_CATEGORIES, ASSET_STATES } from '#lib/api/index.js';
	import type { Asset, AssetCategory, AssetInput, AssetState, User } from '#lib/api/index.js';
	import FormField from '#lib/components/form-field.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { assetCategoryLabels, assetStateLabels } from '#lib/format.js';
	import type { Snippet } from 'svelte';

	let {
		asset,
		submitLabel,
		onsubmit,
		actions
	}: {
		/** Equipo a editar; sin él, el formulario está vacío (alta). */
		asset?: Asset;
		submitLabel: string;
		/** Recibe el objeto completo (el PATCH reemplaza todos los campos). */
		onsubmit: (input: AssetInput) => Promise<void>;
		/** Botones extra junto al de guardar. */
		actions?: Snippet;
	} = $props();

	// Los valores iniciales se leen una sola vez: el formulario es el dueño de lo que se edita.
	/* svelte-ignore state_referenced_locally */
	let tag = $state(asset?.tag ?? '');
	/* svelte-ignore state_referenced_locally */
	let name = $state(asset?.name ?? '');
	/* svelte-ignore state_referenced_locally */
	let category = $state<AssetCategory>(asset?.category ?? 'computer');
	/* svelte-ignore state_referenced_locally */
	let model = $state(asset?.model ?? '');
	/* svelte-ignore state_referenced_locally */
	let serial = $state(asset?.serial ?? '');
	/* svelte-ignore state_referenced_locally */
	let stateValue = $state<AssetState>(asset?.state ?? 'in_stock');
	/* svelte-ignore state_referenced_locally */
	let assignedTo = $state(asset?.assigned_to ? String(asset.assigned_to.id) : '');
	/* svelte-ignore state_referenced_locally */
	let location = $state(asset?.location ?? '');
	/* svelte-ignore state_referenced_locally */
	let purchaseDate = $state(asset?.purchase_date ?? '');
	/* svelte-ignore state_referenced_locally */
	let cost = $state<string | number | null>(asset?.purchase_cost ?? '');
	/* svelte-ignore state_referenced_locally */
	let warrantyUntil = $state(asset?.warranty_until ?? '');
	/* svelte-ignore state_referenced_locally */
	let notes = $state(asset?.notes ?? '');

	let users = $state<User[]>([]);
	let errors = $state<Record<string, string>>({});
	let message = $state<string | null>(null);
	let submitting = $state(false);

	$effect(() => {
		api
			.listUsers()
			.then((r) => (users = r.items))
			.catch(() => (users = []));
	});

	// Si el responsable actual no viene en la lista (p. ej. un usuario desactivado), se conserva.
	const userOptions = $derived(
		asset?.assigned_to && !users.some((u) => u.id === asset.assigned_to?.id)
			? [{ id: asset.assigned_to.id, name: asset.assigned_to.name } as User, ...users]
			: users
	);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		errors = {};
		message = null;
		const next: Record<string, string> = {};
		if (!tag.trim()) next.tag = 'La etiqueta es obligatoria';
		if (!name.trim()) next.name = 'El nombre es obligatorio';
		const rawCost = String(cost ?? '').trim();
		const costValue = rawCost === '' ? null : Number(rawCost);
		if (costValue !== null && (!Number.isFinite(costValue) || costValue < 0)) {
			next.purchase_cost = 'Escribe un monto válido';
		}
		if (Object.keys(next).length) {
			errors = next;
			return;
		}
		submitting = true;
		try {
			await onsubmit({
				tag: tag.trim(),
				name: name.trim(),
				category,
				model: model.trim(),
				serial: serial.trim(),
				state: stateValue,
				assigned_to: assignedTo ? Number(assignedTo) : null,
				location: location.trim(),
				purchase_date: purchaseDate,
				purchase_cost: costValue,
				warranty_until: warrantyUntil,
				notes
			});
		} catch (err) {
			if (err instanceof ApiError && Object.keys(err.fields).length) errors = err.fields;
			else message = err instanceof Error ? err.message : 'No se pudo guardar';
		} finally {
			submitting = false;
		}
	}
</script>

<form class="grid gap-5" onsubmit={submit} novalidate>
	{#if message}
		<Alert.Root variant="destructive">
			<Alert.Description>{message}</Alert.Description>
		</Alert.Root>
	{/if}

	<div class="grid gap-5 sm:grid-cols-2">
		<FormField
			id="asset-tag"
			label="Etiqueta"
			errors={errors.tag ? [errors.tag] : undefined}
			description="Identificador único, p. ej. LAP-001."
		>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					bind:value={tag}
					maxlength={60}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>
		<FormField id="asset-name" label="Nombre" errors={errors.name ? [errors.name] : undefined}>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					bind:value={name}
					placeholder="Ej.: Dell Latitude 5350"
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>
		<FormField
			id="asset-category"
			label="Categoría"
			errors={errors.category ? [errors.category] : undefined}
		>
			{#snippet children({ id, invalid, describedBy })}
				<NativeSelect
					{id}
					class="w-full"
					bind:value={category}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				>
					{#each ASSET_CATEGORIES as c (c)}
						<NativeSelectOption value={c}>{assetCategoryLabels[c]}</NativeSelectOption>
					{/each}
				</NativeSelect>
			{/snippet}
		</FormField>
		<FormField id="asset-model" label="Modelo" errors={errors.model ? [errors.model] : undefined}>
			{#snippet children({ id, invalid, describedBy })}
				<Input {id} bind:value={model} aria-invalid={invalid} aria-describedby={describedBy} />
			{/snippet}
		</FormField>
		<FormField
			id="asset-serial"
			label="Número de serie"
			errors={errors.serial ? [errors.serial] : undefined}
		>
			{#snippet children({ id, invalid, describedBy })}
				<Input {id} bind:value={serial} aria-invalid={invalid} aria-describedby={describedBy} />
			{/snippet}
		</FormField>
		<FormField id="asset-state" label="Estado" errors={errors.state ? [errors.state] : undefined}>
			{#snippet children({ id, invalid, describedBy })}
				<NativeSelect
					{id}
					class="w-full"
					bind:value={stateValue}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				>
					{#each ASSET_STATES as s (s)}
						<NativeSelectOption value={s}>{assetStateLabels[s]}</NativeSelectOption>
					{/each}
				</NativeSelect>
			{/snippet}
		</FormField>
		<FormField
			id="asset-assignee"
			label="Responsable"
			errors={errors.assigned_to ? [errors.assigned_to] : undefined}
			description="Al asignar a alguien, un equipo en almacén pasa a «En uso»."
		>
			{#snippet children({ id, invalid, describedBy })}
				<NativeSelect
					{id}
					class="w-full"
					bind:value={assignedTo}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				>
					<NativeSelectOption value="">Sin asignar</NativeSelectOption>
					{#each userOptions as u (u.id)}
						<NativeSelectOption value={String(u.id)}>{u.name}</NativeSelectOption>
					{/each}
				</NativeSelect>
			{/snippet}
		</FormField>
		<FormField
			id="asset-location"
			label="Ubicación"
			errors={errors.location ? [errors.location] : undefined}
		>
			{#snippet children({ id, invalid, describedBy })}
				<Input {id} bind:value={location} aria-invalid={invalid} aria-describedby={describedBy} />
			{/snippet}
		</FormField>
		<FormField
			id="asset-purchase-date"
			label="Fecha de compra"
			errors={errors.purchase_date ? [errors.purchase_date] : undefined}
		>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					type="date"
					bind:value={purchaseDate}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>
		<FormField
			id="asset-cost"
			label="Costo (MXN)"
			errors={errors.purchase_cost ? [errors.purchase_cost] : undefined}
		>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					type="number"
					min="0"
					step="0.01"
					inputmode="decimal"
					bind:value={cost}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>
		<FormField
			id="asset-warranty"
			label="Garantía hasta"
			errors={errors.warranty_until ? [errors.warranty_until] : undefined}
		>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					type="date"
					bind:value={warrantyUntil}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>
	</div>

	<FormField id="asset-notes" label="Notas" errors={errors.notes ? [errors.notes] : undefined}>
		{#snippet children({ id, invalid, describedBy })}
			<Textarea
				{id}
				rows={4}
				bind:value={notes}
				aria-invalid={invalid}
				aria-describedby={describedBy}
			/>
		{/snippet}
	</FormField>

	<div class="flex flex-wrap gap-2">
		<Button type="submit" disabled={submitting}>
			{submitting ? 'Guardando…' : submitLabel}
		</Button>
		{#if actions}{@render actions()}{/if}
	</div>
</form>
