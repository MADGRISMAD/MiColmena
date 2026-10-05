<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import LockIcon from '@lucide/svelte/icons/lock';
	import { toast } from 'svelte-sonner';
	import {
		api,
		ApiError,
		TICKET_PRIORITIES,
		TICKET_STATUSES,
		type Comment,
		type Ticket,
		type UpdateTicketInput,
		type User
	} from '#lib/api/index.js';
	import FormField from '#lib/components/form-field.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { formatDateTime, formatRelative, priorityLabels, statusLabels } from '#lib/format.js';
	import { apiForm } from '#lib/forms.js';
	import { commentSchema, emptyComment } from '#lib/schemas.js';
	import { isStaff } from '#lib/stores/auth.js';
	import { cn } from '#lib/utils.js';

	const ticketId = $derived(Number(page.params.id));

	let ticket = $state<Ticket | null>(null);
	let comments = $state<Comment[]>([]);
	let agents = $state<User[]>([]);
	let error = $state<string | null>(null);
	let saving = $state(false);

	async function load(id: number) {
		error = null;
		ticket = null;
		try {
			const [t, c] = await Promise.all([api.getTicket(id), api.listComments(id)]);
			ticket = t;
			comments = c.items;
		} catch (err) {
			error =
				err instanceof ApiError && err.status === 404
					? 'Este ticket no existe o no tienes acceso a él.'
					: err instanceof Error
						? err.message
						: 'No se pudo cargar el ticket';
		}
	}

	$effect(() => {
		load(ticketId);
	});

	$effect(() => {
		if (!$isStaff) return;
		Promise.all([api.listUsers('agent'), api.listUsers('admin')])
			.then(([a, b]) => (agents = [...b.items, ...a.items]))
			.catch(() => (agents = []));
	});

	async function update(changes: UpdateTicketInput, successMessage: string) {
		if (!ticket) return;
		saving = true;
		try {
			ticket = await api.updateTicket(ticket.id, changes);
			toast.success(successMessage);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo guardar el cambio');
		} finally {
			saving = false;
		}
	}

	const { form, errors, message, submitting, enhance } = apiForm(
		commentSchema,
		emptyComment,
		async (data) => {
			const comment = await api.addComment(ticketId, data.body, $isStaff && data.internal);
			comments = [...comments, comment];
			// Responder puede reabrir el ticket en el servidor; se recarga para reflejarlo.
			ticket = await api.getTicket(ticketId);
			return emptyComment;
		},
		{ id: 'comment' }
	);

	const isClosed = $derived(ticket?.status === 'closed');
</script>

<a
	href={resolve('/(app)/tickets')}
	class="mb-4 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
>
	<ArrowLeftIcon class="size-4" aria-hidden="true" />
	Volver a tickets
</a>

{#if error}
	<Card.Root>
		<Card.Content class="py-10 text-center">
			<p class="font-medium">{error}</p>
			<Button class="mt-4" variant="outline" href={resolve('/(app)/tickets')}
				>Ver mis tickets</Button
			>
		</Card.Content>
	</Card.Root>
{:else if !ticket}
	<div class="grid gap-4" aria-busy="true">
		<Skeleton class="h-10 w-2/3" />
		<Skeleton class="h-40" />
	</div>
{:else}
	<PageHeader title={`#${ticket.id} · ${ticket.title}`}>
		{#snippet actions()}
			{#if !$isStaff && !isClosed}
				<Button
					variant="outline"
					disabled={saving}
					onclick={() => update({ status: 'closed' }, 'Ticket cerrado')}
				>
					Cerrar ticket
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	<div class="grid gap-6 lg:grid-cols-[1fr_18rem]">
		<div class="grid min-w-0 content-start gap-6">
			<Card.Root>
				<Card.Header>
					<Card.Description>
						{ticket.requester.name} abrió este ticket {formatRelative(ticket.created_at)}
					</Card.Description>
				</Card.Header>
				<Card.Content>
					{#if ticket.description}
						<p class="whitespace-pre-wrap">{ticket.description}</p>
					{:else}
						<p class="text-muted-foreground italic">Sin descripción.</p>
					{/if}
				</Card.Content>
			</Card.Root>

			<section aria-labelledby="comments-title" class="grid gap-3">
				<h2 id="comments-title" class="font-semibold">
					Conversación ({comments.length})
				</h2>

				{#each comments as comment (comment.id)}
					<Card.Root
						class={cn(comment.internal && 'border-amber-300 bg-amber-50 dark:bg-amber-950/30')}
					>
						<Card.Content>
							<div class="mb-2 flex flex-wrap items-center gap-2 text-sm">
								<span class="font-medium">{comment.author.name}</span>
								<time
									class="text-muted-foreground"
									datetime={comment.created_at}
									title={formatDateTime(comment.created_at)}
								>
									{formatRelative(comment.created_at)}
								</time>
								{#if comment.internal}
									<span
										class="inline-flex items-center gap-1 text-xs text-amber-700 dark:text-amber-400"
									>
										<LockIcon class="size-3" aria-hidden="true" />
										Nota interna
									</span>
								{/if}
							</div>
							<p class="whitespace-pre-wrap">{comment.body}</p>
						</Card.Content>
					</Card.Root>
				{:else}
					<p class="text-sm text-muted-foreground">Todavía no hay respuestas.</p>
				{/each}

				{#if isClosed}
					<p class="text-sm text-muted-foreground">
						Este ticket está cerrado. Abre uno nuevo si necesitas más ayuda.
					</p>
				{:else}
					<form method="POST" use:enhance class="grid gap-3" novalidate>
						{#if $message}
							<Alert.Root variant="destructive">
								<Alert.Description>{$message}</Alert.Description>
							</Alert.Root>
						{/if}
						<FormField id="body" label="Responder" errors={$errors.body}>
							{#snippet children({ id, invalid, describedBy })}
								<Textarea
									{id}
									name="body"
									rows={4}
									placeholder="Escribe tu respuesta…"
									bind:value={$form.body}
									aria-invalid={invalid}
									aria-describedby={describedBy}
								/>
							{/snippet}
						</FormField>
						<div class="flex flex-wrap items-center justify-between gap-3">
							{#if $isStaff}
								<label class="flex items-center gap-2 text-sm">
									<input
										type="checkbox"
										name="internal"
										bind:checked={$form.internal}
										class="size-4 accent-primary"
									/>
									Nota interna (el cliente no la ve)
								</label>
							{:else}
								<span></span>
							{/if}
							<Button type="submit" disabled={$submitting}>
								{$submitting ? 'Enviando…' : $form.internal ? 'Añadir nota' : 'Enviar respuesta'}
							</Button>
						</div>
					</form>
				{/if}
			</section>
		</div>

		<Card.Root class="content-start self-start">
			<Card.Content class="grid gap-4 text-sm">
				{#if $isStaff}
					<div class="grid gap-1.5">
						<Label for="status">Estado</Label>
						<NativeSelect
							id="status"
							class="w-full"
							value={ticket.status}
							disabled={saving}
							onchange={(e) => {
								const status = e.currentTarget.value as Ticket['status'];
								update({ status }, `Estado: ${statusLabels[status]}`);
							}}
						>
							{#each TICKET_STATUSES as status (status)}
								<NativeSelectOption value={status}>{statusLabels[status]}</NativeSelectOption>
							{/each}
						</NativeSelect>
					</div>
					<div class="grid gap-1.5">
						<Label for="priority">Prioridad</Label>
						<NativeSelect
							id="priority"
							class="w-full"
							value={ticket.priority}
							disabled={saving}
							onchange={(e) => {
								const priority = e.currentTarget.value as Ticket['priority'];
								update({ priority }, `Prioridad: ${priorityLabels[priority]}`);
							}}
						>
							{#each TICKET_PRIORITIES as priority (priority)}
								<NativeSelectOption value={priority}>{priorityLabels[priority]}</NativeSelectOption>
							{/each}
						</NativeSelect>
					</div>
					<div class="grid gap-1.5">
						<Label for="assignee">Asignado a</Label>
						<NativeSelect
							id="assignee"
							class="w-full"
							value={ticket.assignee ? String(ticket.assignee.id) : ''}
							disabled={saving}
							onchange={(e) => {
								const value = e.currentTarget.value;
								update(
									{ assignee_id: value ? Number(value) : null },
									value ? 'Ticket asignado' : 'Ticket sin asignar'
								);
							}}
						>
							<NativeSelectOption value="">Sin asignar</NativeSelectOption>
							{#each agents as agent (agent.id)}
								<NativeSelectOption value={String(agent.id)}>{agent.name}</NativeSelectOption>
							{/each}
						</NativeSelect>
					</div>
				{:else}
					<div class="flex items-center justify-between">
						<span class="text-muted-foreground">Estado</span>
						<StatusBadge status={ticket.status} />
					</div>
					<div class="flex items-center justify-between">
						<span class="text-muted-foreground">Prioridad</span>
						<PriorityBadge priority={ticket.priority} />
					</div>
					<div class="flex items-center justify-between">
						<span class="text-muted-foreground">Asignado a</span>
						<span>{ticket.assignee?.name ?? 'Pendiente'}</span>
					</div>
				{/if}

				<dl class="grid gap-2 border-t pt-4">
					<div class="flex justify-between gap-2">
						<dt class="text-muted-foreground">Solicitante</dt>
						<dd class="truncate">{ticket.requester.name}</dd>
					</div>
					{#if ticket.category}
						<div class="flex justify-between gap-2">
							<dt class="text-muted-foreground">Categoría</dt>
							<dd class="truncate">{ticket.category}</dd>
						</div>
					{/if}
					<div class="flex justify-between gap-2">
						<dt class="text-muted-foreground">Creado</dt>
						<dd>{formatDateTime(ticket.created_at)}</dd>
					</div>
					<div class="flex justify-between gap-2">
						<dt class="text-muted-foreground">Actualizado</dt>
						<dd>{formatRelative(ticket.updated_at)}</dd>
					</div>
					{#if ticket.resolved_at}
						<div class="flex justify-between gap-2">
							<dt class="text-muted-foreground">Resuelto</dt>
							<dd>{formatDateTime(ticket.resolved_at)}</dd>
						</div>
					{/if}
				</dl>
			</Card.Content>
		</Card.Root>
	</div>
{/if}
