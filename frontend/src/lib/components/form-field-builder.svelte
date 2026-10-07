<script lang="ts" module>
	import type { FieldType, FormField } from '#lib/api/index.js';

	/** Un campo en edición: las opciones se escriben una por línea. */
	export interface EditableField {
		uid: number;
		key?: string;
		label: string;
		type: FieldType;
		required: boolean;
		optionsText: string;
	}

	let seq = 0;

	export function newField(): EditableField {
		return { uid: ++seq, label: '', type: 'text', required: false, optionsText: '' };
	}

	export function toEditable(fields: FormField[]): EditableField[] {
		return fields.map((f) => ({
			uid: ++seq,
			key: f.key,
			label: f.label,
			type: f.type,
			required: f.required,
			optionsText: (f.options ?? []).join('\n')
		}));
	}

	/** Campos listos para la API: los nuevos van sin `key` y solo las listas llevan opciones. */
	export function toPayload(fields: EditableField[]) {
		return fields.map((f) => ({
			...(f.key ? { key: f.key } : {}),
			label: f.label.trim(),
			type: f.type,
			required: f.required,
			...(f.type === 'select'
				? {
						options: f.optionsText
							.split('\n')
							.map((o) => o.trim())
							.filter(Boolean)
					}
				: {})
		}));
	}

	export const fieldTypeLabels: Record<FieldType, string> = {
		text: 'Texto',
		textarea: 'Texto largo',
		select: 'Lista',
		number: 'Número',
		date: 'Fecha'
	};
</script>

<script lang="ts">
	import ArrowDownIcon from '@lucide/svelte/icons/arrow-down';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import { FIELD_TYPES } from '#lib/api/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';

	let {
		fields = $bindable(),
		idPrefix,
		error
	}: { fields: EditableField[]; idPrefix: string; error?: string } = $props();

	function move(i: number, delta: number) {
		const j = i + delta;
		if (j < 0 || j >= fields.length) return;
		const next = [...fields];
		[next[i], next[j]] = [next[j], next[i]];
		fields = next;
	}
</script>

<fieldset class="grid gap-3">
	<legend class="text-sm font-medium">Preguntas del formulario</legend>
	{#if fields.length === 0}
		<p class="text-sm text-muted-foreground">Sin preguntas: el usuario solo enviará notas.</p>
	{/if}
	{#each fields as field, i (field.uid)}
		<div
			class="grid gap-3 rounded-lg border bg-muted/30 p-3"
			role="group"
			aria-label={`Campo ${i + 1}`}
		>
			<div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_10rem]">
				<div class="grid gap-1.5">
					<Label for={`${idPrefix}-label-${field.uid}`}>Etiqueta</Label>
					<Input id={`${idPrefix}-label-${field.uid}`} bind:value={field.label} />
				</div>
				<div class="grid gap-1.5">
					<Label for={`${idPrefix}-type-${field.uid}`}>Tipo</Label>
					<NativeSelect id={`${idPrefix}-type-${field.uid}`} class="w-full" bind:value={field.type}>
						{#each FIELD_TYPES as type (type)}
							<NativeSelectOption value={type}>{fieldTypeLabels[type]}</NativeSelectOption>
						{/each}
					</NativeSelect>
				</div>
			</div>
			{#if field.type === 'select'}
				<div class="grid gap-1.5">
					<Label for={`${idPrefix}-opts-${field.uid}`}>Opciones (una por línea)</Label>
					<Textarea id={`${idPrefix}-opts-${field.uid}`} rows={3} bind:value={field.optionsText} />
				</div>
			{/if}
			<div class="flex flex-wrap items-center justify-between gap-2">
				<label class="flex items-center gap-2 text-sm">
					<input type="checkbox" class="size-4 accent-amber-500" bind:checked={field.required} />
					Obligatorio
				</label>
				<div class="flex gap-1">
					<Button
						type="button"
						variant="ghost"
						size="icon-sm"
						aria-label={`Subir campo ${i + 1}`}
						disabled={i === 0}
						onclick={() => move(i, -1)}
					>
						<ArrowUpIcon aria-hidden="true" />
					</Button>
					<Button
						type="button"
						variant="ghost"
						size="icon-sm"
						aria-label={`Bajar campo ${i + 1}`}
						disabled={i === fields.length - 1}
						onclick={() => move(i, 1)}
					>
						<ArrowDownIcon aria-hidden="true" />
					</Button>
					<Button
						type="button"
						variant="ghost"
						size="icon-sm"
						aria-label={`Quitar campo ${i + 1}`}
						onclick={() => (fields = fields.filter((f) => f.uid !== field.uid))}
					>
						<Trash2Icon aria-hidden="true" />
					</Button>
				</div>
			</div>
		</div>
	{/each}
	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{/if}
	<div>
		<Button
			type="button"
			variant="outline"
			size="sm"
			onclick={() => (fields = [...fields, newField()])}
		>
			<PlusIcon aria-hidden="true" />
			Añadir campo
		</Button>
	</div>
</fieldset>
