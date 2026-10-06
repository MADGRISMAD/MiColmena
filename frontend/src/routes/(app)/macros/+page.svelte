<script lang="ts">
	import LockIcon from '@lucide/svelte/icons/lock';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import { toast } from 'svelte-sonner';
	import { api, ApiError, TICKET_STATUSES, type Macro, type MacroInput } from '#lib/api/index.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { statusLabels } from '#lib/format.js';

	let macros = $state<Macro[] | null>(null);
	/** null = sin formulario abierto; 0 = nueva; otro = editando ese id. */
	let editing = $state<number | null>(null);
	let draft = $state<MacroInput>({ title: '', body: '', status: '', internal: false });
	let fieldErrors = $state<Record<string, string>>({});
	let saving = $state(false);

	async function load() {
		macros = await api
			.listMacros()
			.then((r) => r.items)
			.catch(() => []);
	}
	$effect(() => {
		load();
	});

	function edit(macro?: Macro) {
		fieldErrors = {};
		editing = macro?.id ?? 0;
		draft = macro
			? { title: macro.title, body: macro.body, status: macro.status, internal: macro.internal }
			: { title: '', body: '', status: '', internal: false };
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		saving = true;
		fieldErrors = {};
		try {
			if (editing) await api.updateMacro(editing, draft);
			else await api.createMacro(draft);
			toast.success('Respuesta guardada');
			editing = null;
			await load();
		} catch (err) {
			if (err instanceof ApiError && Object.keys(err.fields).length) fieldErrors = err.fields;
			else toast.error(err instanceof Error ? err.message : 'No se pudo guardar');
		} finally {
			saving = false;
		}
	}

	async function remove(macro: Macro) {
		if (!confirm(`¿Borrar la respuesta «${macro.title}»?`)) return;
		try {
			await api.deleteMacro(macro.id);
			macros = macros?.filter((m) => m.id !== macro.id) ?? null;
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo borrar');
		}
	}
</script>

<PageHeader
	eyebrow="Recursos"
	title="Respuestas guardadas"
	description="Textos listos para las preguntas de siempre. Insértalos desde el editor de cualquier ticket."
>
	{#snippet actions()}
		<Button onclick={() => edit()}>
			<PlusIcon aria-hidden="true" />
			Nueva respuesta
		</Button>
	{/snippet}
</PageHeader>

{#snippet editor()}
	<Card.Root class="border-honey/60">
		<Card.Content>
			<form class="grid gap-4" onsubmit={save} novalidate>
				<div class="grid gap-1.5">
					<Label for="macro-title">Título</Label>
					<Input
						id="macro-title"
						bind:value={draft.title}
						placeholder="Por ejemplo: Reiniciar el router"
						aria-invalid={!!fieldErrors.title}
					/>
					{#if fieldErrors.title}<p class="text-sm text-destructive">{fieldErrors.title}</p>{/if}
				</div>
				<div class="grid gap-1.5">
					<Label for="macro-body">Texto</Label>
					<Textarea
						id="macro-body"
						bind:value={draft.body}
						rows={6}
						aria-invalid={!!fieldErrors.body}
						aria-describedby="macro-vars"
					/>
					<p id="macro-vars" class="text-xs text-muted-foreground">
						Variables: <code>{'{{solicitante}}'}</code>, <code>{'{{agente}}'}</code> y
						<code>{'{{ticket}}'}</code> se reemplazan al insertarla.
					</p>
					{#if fieldErrors.body}<p class="text-sm text-destructive">{fieldErrors.body}</p>{/if}
				</div>
				<div class="flex flex-wrap items-end gap-4">
					<div class="grid gap-1.5">
						<Label for="macro-status">Al enviar, cambiar el estado a</Label>
						<NativeSelect id="macro-status" bind:value={draft.status}>
							<NativeSelectOption value="">No cambiar</NativeSelectOption>
							{#each TICKET_STATUSES as status (status)}
								<NativeSelectOption value={status}>{statusLabels[status]}</NativeSelectOption>
							{/each}
						</NativeSelect>
					</div>
					<label class="flex items-center gap-2 pb-2 text-sm">
						<input type="checkbox" class="size-4 accent-amber-500" bind:checked={draft.internal} />
						Es una nota interna
					</label>
				</div>
				<div class="flex gap-2">
					<Button type="submit" disabled={saving}>{saving ? 'Guardando…' : 'Guardar'}</Button>
					<Button type="button" variant="ghost" onclick={() => (editing = null)}>Cancelar</Button>
				</div>
			</form>
		</Card.Content>
	</Card.Root>
{/snippet}

<div class="grid gap-3">
	{#if editing === 0}{@render editor()}{/if}

	{#if macros === null}
		<Skeleton class="h-24" />
		<Skeleton class="h-24" />
	{:else if macros.length === 0 && editing === null}
		<Card.Root>
			<EmptyState
				title="Aún no hay respuestas guardadas"
				description="Crea la primera para responder en segundos."
			>
				<Button onclick={() => edit()}>Nueva respuesta</Button>
			</EmptyState>
		</Card.Root>
	{:else}
		{#each macros as macro (macro.id)}
			{#if editing === macro.id}
				{@render editor()}
			{:else}
				<Card.Root class="py-4">
					<Card.Content class="flex-row items-start gap-4 px-5">
						<div class="min-w-0 flex-1">
							<p class="flex flex-wrap items-center gap-2 font-medium">
								{macro.title}
								{#if macro.internal}
									<span
										class="inline-flex items-center gap-1 rounded bg-amber-100 px-1.5 text-xs text-amber-800 dark:bg-amber-500/15 dark:text-amber-300"
									>
										<LockIcon class="size-3" aria-hidden="true" /> Nota interna
									</span>
								{/if}
								{#if macro.status}
									<span class="rounded bg-muted px-1.5 text-xs text-muted-foreground">
										→ {statusLabels[macro.status]}
									</span>
								{/if}
							</p>
							<p class="mt-1 line-clamp-2 text-sm whitespace-pre-wrap text-muted-foreground">
								{macro.body}
							</p>
						</div>
						<div class="flex shrink-0 gap-1">
							<Button
								variant="ghost"
								size="icon"
								onclick={() => edit(macro)}
								aria-label={`Editar ${macro.title}`}
							>
								<PencilIcon aria-hidden="true" />
							</Button>
							<Button
								variant="ghost"
								size="icon"
								onclick={() => remove(macro)}
								aria-label={`Borrar ${macro.title}`}
							>
								<Trash2Icon aria-hidden="true" />
							</Button>
						</div>
					</Card.Content>
				</Card.Root>
			{/if}
		{/each}
	{/if}
</div>
