<script lang="ts">
	import { goto } from '$app/navigation';
	import BookOpenIcon from '@lucide/svelte/icons/book-open';
	import LightbulbIcon from '@lucide/svelte/icons/lightbulb';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import { api, TICKET_PRIORITIES, type Article } from '#lib/api/index.js';
	import FilePicker from '#lib/components/file-picker.svelte';
	import FormField from '#lib/components/form-field.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { priorityLabels } from '#lib/format.js';
	import { apiForm } from '#lib/forms.js';
	import { emptyTicket, ticketSchema } from '#lib/schemas.js';

	let files = $state<File[]>([]);
	let categories = $state<string[]>([]);
	let suggestions = $state<Article[]>([]);

	$effect(() => {
		api
			.listCategories()
			.then((r) => (categories = r.items.map((c) => c.name)))
			.catch(() => {});
	});

	const { form, errors, message, submitting, enhance } = apiForm(
		ticketSchema,
		emptyTicket,
		async (data) => {
			const ticket = await api.createTicket(data);
			const failed: string[] = [];
			for (const file of files) {
				await api.uploadAttachment(ticket.id, file).catch(() => failed.push(file.name));
			}
			if (failed.length) toast.error(`No se pudieron adjuntar: ${failed.join(', ')}`);
			toast.success(`Ticket #${ticket.id} creado`);
			await goto(resolve('/(app)/tickets/[id]', { id: String(ticket.id) }));
		}
	);

	// Mientras escribe el título, se sugieren artículos de ayuda que podrían resolverlo.
	$effect(() => {
		const q = $form.title.trim();
		if (q.length < 4) {
			suggestions = [];
			return;
		}
		const timer = setTimeout(() => {
			api
				.listArticles(q)
				.then((r) => (suggestions = r.items.filter((a) => a.published).slice(0, 3)))
				.catch(() => {});
		}, 350);
		return () => clearTimeout(timer);
	});
</script>

<PageHeader
	eyebrow="Soporte"
	title="Nuevo ticket"
	description="Cuéntanos qué pasa y te ayudaremos lo antes posible."
/>

<div class="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_18rem]">
	<Card.Root>
		<Card.Content>
			<form method="POST" use:enhance class="grid gap-5" novalidate>
				{#if $message}
					<Alert.Root variant="destructive">
						<Alert.Description>{$message}</Alert.Description>
					</Alert.Root>
				{/if}

				<FormField id="title" label="Título" errors={$errors.title}>
					{#snippet children({ id, invalid, describedBy })}
						<Input
							{id}
							name="title"
							placeholder="Ej.: No puedo acceder a mi cuenta"
							bind:value={$form.title}
							aria-invalid={invalid}
							aria-describedby={describedBy}
						/>
					{/snippet}
				</FormField>

				<FormField
					id="description"
					label="Descripción"
					errors={$errors.description}
					description="Incluye los pasos para reproducir el problema y cualquier mensaje de error."
				>
					{#snippet children({ id, invalid, describedBy })}
						<Textarea
							{id}
							name="description"
							rows={6}
							bind:value={$form.description}
							aria-invalid={invalid}
							aria-describedby={describedBy}
						/>
					{/snippet}
				</FormField>

				<div class="grid gap-5 sm:grid-cols-2">
					<FormField id="priority" label="Prioridad" errors={$errors.priority}>
						{#snippet children({ id, invalid, describedBy })}
							<NativeSelect
								{id}
								name="priority"
								class="w-full"
								bind:value={$form.priority}
								aria-invalid={invalid}
								aria-describedby={describedBy}
							>
								{#each TICKET_PRIORITIES as priority (priority)}
									<NativeSelectOption value={priority}
										>{priorityLabels[priority]}</NativeSelectOption
									>
								{/each}
							</NativeSelect>
						{/snippet}
					</FormField>

					{#if categories.length > 0}
						<FormField id="category" label="Categoría" errors={$errors.category}>
							{#snippet children({ id, invalid, describedBy })}
								<NativeSelect
									{id}
									name="category"
									class="w-full"
									bind:value={$form.category}
									aria-invalid={invalid}
									aria-describedby={describedBy}
								>
									<NativeSelectOption value="">Sin categoría</NativeSelectOption>
									{#each categories as category (category)}
										<NativeSelectOption value={category}>{category}</NativeSelectOption>
									{/each}
								</NativeSelect>
							{/snippet}
						</FormField>
					{/if}
				</div>

				<div class="flex flex-wrap items-center gap-2 rounded-lg border border-dashed p-2">
					<FilePicker bind:files disabled={$submitting} />
					{#if files.length === 0}
						<span class="text-xs text-muted-foreground"
							>Capturas de pantalla o documentos, hasta 10 MB cada uno.</span
						>
					{/if}
				</div>

				<div class="flex gap-2">
					<Button type="submit" disabled={$submitting}>
						{$submitting ? 'Creando…' : 'Crear ticket'}
					</Button>
					<Button variant="ghost" href={resolve('/(app)/tickets')}>Cancelar</Button>
				</div>
			</form>
		</Card.Content>
	</Card.Root>

	<div class="grid gap-4">
		{#if suggestions.length > 0}
			<aside class="rounded-xl border border-honey/60 bg-card p-5" aria-live="polite">
				<h2 class="flex items-center gap-2 font-semibold">
					<BookOpenIcon class="size-4 text-amber-600" aria-hidden="true" />
					¿Te sirve alguno de estos?
				</h2>
				<ul class="mt-3 grid gap-2 text-sm">
					{#each suggestions as a (a.id)}
						<li>
							<a
								href={resolve('/help/[id]', { id: String(a.id) })}
								target="_blank"
								rel="noopener"
								class="text-primary underline-offset-4 hover:underline dark:text-honey"
							>
								{a.title}
							</a>
						</li>
					{/each}
				</ul>
			</aside>
		{/if}
		<aside class="bg-honeycomb relative overflow-hidden rounded-xl border bg-card p-5">
			<div class="flex items-center gap-2">
				<span class="hex grid size-8 place-items-center bg-honey text-honey-foreground">
					<LightbulbIcon class="size-4" aria-hidden="true" />
				</span>
				<h2 class="font-semibold">Para que te ayudemos más rápido</h2>
			</div>
			<ul class="mt-4 grid gap-3 text-sm text-muted-foreground">
				<li>
					<span class="font-medium text-foreground">Un título concreto.</span> «No puedo exportar a PDF»
					mejor que «Error».
				</li>
				<li>
					<span class="font-medium text-foreground">Qué esperabas y qué pasó.</span> Incluye el mensaje
					de error exacto si lo hay.
				</li>
				<li>
					<span class="font-medium text-foreground">Los pasos para reproducirlo.</span> Aunque parezcan
					obvios.
				</li>
				<li>
					<span class="font-medium text-foreground">La prioridad real.</span> «Urgente» es cuando algo
					importante está detenido.
				</li>
			</ul>
		</aside>
	</div>
</div>
