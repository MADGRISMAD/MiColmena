<script lang="ts">
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import {
		api,
		ApiError,
		TICKET_PRIORITIES,
		type ServiceCategory,
		type ServiceItem,
		type TicketPriority
	} from '#lib/api/index.js';
	import FormField from '#lib/components/form-field.svelte';
	import FormFieldBuilder, {
		toEditable,
		toPayload,
		type EditableField
	} from '#lib/components/form-field-builder.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { priorityLabels } from '#lib/format.js';

	let {
		item = null,
		categoryId,
		categories,
		ticketCategories,
		onsaved,
		oncancel
	}: {
		item?: ServiceItem | null;
		categoryId: number;
		categories: ServiceCategory[];
		ticketCategories: string[];
		onsaved: () => void;
		oncancel: () => void;
	} = $props();

	const uid = $props.id();
	// El formulario parte de los valores iniciales y no sigue a la prop.
	const seed = untrack(() => ({ item, categoryId }));

	let name = $state(seed.item?.name ?? '');
	let description = $state(seed.item?.description ?? '');
	let category = $state(String(seed.item?.category_id ?? seed.categoryId));
	let priority = $state<TicketPriority>(seed.item?.priority ?? 'medium');
	let ticketCategory = $state(seed.item?.ticket_category ?? '');
	let active = $state(seed.item?.active ?? true);
	let fields = $state<EditableField[]>(toEditable(seed.item?.fields ?? []));
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
		const input = {
			category_id: Number(category),
			name: name.trim(),
			description: description.trim(),
			fields: toPayload(fields),
			priority,
			ticket_category: ticketCategory,
			active
		};
		saving = true;
		try {
			if (item) await api.updateServiceItem(item.id, input);
			else await api.createServiceItem(input);
			toast.success(item ? 'Servicio actualizado' : 'Servicio creado');
			onsaved();
		} catch (e) {
			if (e instanceof ApiError && Object.keys(e.fields).length > 0) errors = e.fields;
			else formError = e instanceof Error ? e.message : 'No se pudo guardar el servicio';
		} finally {
			saving = false;
		}
	}
</script>

<form onsubmit={save} class="grid gap-4 rounded-lg border border-honey/50 bg-card p-4" novalidate>
	<h3 class="font-semibold">{item ? 'Editar servicio' : 'Nuevo servicio'}</h3>
	{#if formError}
		<Alert.Root variant="destructive"><Alert.Description>{formError}</Alert.Description></Alert.Root
		>
	{/if}

	<FormField id={`${uid}-name`} label="Nombre del servicio" errors={err('name')}>
		{#snippet children({ id, invalid, describedBy })}
			<Input {id} bind:value={name} aria-invalid={invalid} aria-describedby={describedBy} />
		{/snippet}
	</FormField>
	<FormField id={`${uid}-desc`} label="Descripción" errors={err('description')}>
		{#snippet children({ id, invalid, describedBy })}
			<Textarea
				{id}
				rows={2}
				bind:value={description}
				aria-invalid={invalid}
				aria-describedby={describedBy}
			/>
		{/snippet}
	</FormField>

	<div class="grid gap-4 sm:grid-cols-3">
		<FormField id={`${uid}-cat`} label="Categoría del catálogo" errors={err('category_id')}>
			{#snippet children({ id, invalid, describedBy })}
				<NativeSelect
					{id}
					class="w-full"
					bind:value={category}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				>
					{#each categories as c (c.id)}
						<NativeSelectOption value={String(c.id)}>{c.name}</NativeSelectOption>
					{/each}
				</NativeSelect>
			{/snippet}
		</FormField>
		<FormField id={`${uid}-prio`} label="Prioridad por defecto" errors={err('priority')}>
			{#snippet children({ id, invalid, describedBy })}
				<NativeSelect
					{id}
					class="w-full"
					bind:value={priority}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				>
					{#each TICKET_PRIORITIES as p (p)}
						<NativeSelectOption value={p}>{priorityLabels[p]}</NativeSelectOption>
					{/each}
				</NativeSelect>
			{/snippet}
		</FormField>
		<FormField id={`${uid}-tcat`} label="Categoría del ticket" errors={err('ticket_category')}>
			{#snippet children({ id, invalid, describedBy })}
				<NativeSelect
					{id}
					class="w-full"
					bind:value={ticketCategory}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				>
					<NativeSelectOption value="">Sin categoría</NativeSelectOption>
					{#each ticketCategories as c (c)}
						<NativeSelectOption value={c}>{c}</NativeSelectOption>
					{/each}
				</NativeSelect>
			{/snippet}
		</FormField>
	</div>

	<label class="flex items-center gap-2 text-sm">
		<input type="checkbox" class="size-4 accent-amber-500" bind:checked={active} />
		Activo
	</label>

	<FormFieldBuilder bind:fields idPrefix={uid} error={errors.fields} />

	<div class="flex gap-2">
		<Button type="submit" disabled={saving}>{saving ? 'Guardando…' : 'Guardar servicio'}</Button>
		<Button type="button" variant="ghost" onclick={oncancel}>Cancelar</Button>
	</div>
</form>
