<script lang="ts">
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { api, ApiError, CATALOG_ICONS, type ServiceCategory } from '#lib/api/index.js';
	import { catalogIconLabels } from '#lib/catalog-icons.js';
	import FormField from '#lib/components/form-field.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';

	let {
		category = null,
		onsaved,
		oncancel
	}: { category?: ServiceCategory | null; onsaved: () => void; oncancel: () => void } = $props();

	const uid = $props.id();
	// El formulario parte de los valores iniciales y no sigue a la prop.
	const seed = untrack(() => ({ category }));

	let name = $state(seed.category?.name ?? '');
	let description = $state(seed.category?.description ?? '');
	let icon = $state<ServiceCategory['icon']>(seed.category?.icon ?? 'package');
	let errors = $state<Record<string, string>>({});
	let formError = $state<string | null>(null);
	let saving = $state(false);

	const err = (key: string) => (errors[key] ? [errors[key]] : undefined);

	async function save(event: SubmitEvent) {
		event.preventDefault();
		errors = {};
		formError = null;
		if (!name.trim()) {
			errors = { name: 'Este campo es obligatorio' };
			return;
		}
		const input = { name: name.trim(), description: description.trim(), icon };
		saving = true;
		try {
			if (category) await api.updateServiceCategory(category.id, input);
			else await api.createServiceCategory(input);
			toast.success(category ? 'Categoría actualizada' : 'Categoría creada');
			onsaved();
		} catch (e) {
			if (e instanceof ApiError && Object.keys(e.fields).length > 0) errors = e.fields;
			else formError = e instanceof Error ? e.message : 'No se pudo guardar la categoría';
		} finally {
			saving = false;
		}
	}
</script>

<form onsubmit={save} class="grid gap-4 rounded-lg border border-honey/50 bg-card p-4" novalidate>
	<h3 class="font-semibold">{category ? 'Editar categoría' : 'Nueva categoría'}</h3>
	{#if formError}
		<Alert.Root variant="destructive"><Alert.Description>{formError}</Alert.Description></Alert.Root
		>
	{/if}
	<div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_12rem]">
		<FormField id={`${uid}-name`} label="Nombre de la categoría" errors={err('name')}>
			{#snippet children({ id, invalid, describedBy })}
				<Input {id} bind:value={name} aria-invalid={invalid} aria-describedby={describedBy} />
			{/snippet}
		</FormField>
		<FormField id={`${uid}-icon`} label="Icono" errors={err('icon')}>
			{#snippet children({ id, invalid, describedBy })}
				<NativeSelect
					{id}
					class="w-full"
					bind:value={icon}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				>
					{#each CATALOG_ICONS as i (i)}
						<NativeSelectOption value={i}>{catalogIconLabels[i]}</NativeSelectOption>
					{/each}
				</NativeSelect>
			{/snippet}
		</FormField>
	</div>
	<FormField id={`${uid}-desc`} label="Descripción" errors={err('description')}>
		{#snippet children({ id, invalid, describedBy })}
			<Input {id} bind:value={description} aria-invalid={invalid} aria-describedby={describedBy} />
		{/snippet}
	</FormField>
	<div class="flex gap-2">
		<Button type="submit" disabled={saving}>{saving ? 'Guardando…' : 'Guardar categoría'}</Button>
		<Button type="button" variant="ghost" onclick={oncancel}>Cancelar</Button>
	</div>
</form>
