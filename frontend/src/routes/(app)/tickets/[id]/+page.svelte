<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import CircleCheckIcon from '@lucide/svelte/icons/circle-check';
	import LockIcon from '@lucide/svelte/icons/lock';
	import MessageSquareIcon from '@lucide/svelte/icons/message-square';
	import RotateCcwIcon from '@lucide/svelte/icons/rotate-ccw';
	import SendIcon from '@lucide/svelte/icons/send';
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
	import EmptyState from '#lib/components/empty-state.svelte';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatDateTime, formatRelative, priorityLabels, statusLabels } from '#lib/format.js';
	import { apiForm } from '#lib/forms.js';
	import { commentSchema, emptyComment } from '#lib/schemas.js';
	import { isStaff, user } from '#lib/stores/auth.js';
	import { cn } from '#lib/utils.js';

	const ticketId = $derived(Number(page.params.id));

	let ticket = $state<Ticket | null>(null);
	let comments = $state<Comment[]>([]);
	let agents = $state<User[]>([]);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let formEl = $state<HTMLFormElement | null>(null);

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
			// Tras enviar se vuelve a "Respuesta pública": así nunca queda una nota interna a medio camino.
			return emptyComment;
		},
		{ id: 'comment' }
	);

	// Ctrl+Enter (o Cmd+Enter) envía, como en las herramientas de soporte.
	function onComposerKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
			event.preventDefault();
			formEl?.requestSubmit();
		}
	}

	const isClosed = $derived(ticket?.status === 'closed');
	const isDone = $derived(ticket?.status === 'resolved' || ticket?.status === 'closed');
	const assignedToMe = $derived(!!ticket?.assignee && ticket.assignee.id === $user?.id);
	const internal = $derived($isStaff && $form.internal);

	function authorRole(authorId: number) {
		return authorId === ticket?.requester.id ? 'Solicitante' : 'Soporte';
	}
</script>

<svelte:head>
	<title>{ticket ? `#${ticket.id} ${ticket.title}` : 'Ticket'} · MiColmena</title>
</svelte:head>

{#if error}
	<Card.Root>
		<EmptyState title={error}>
			<Button variant="outline" href={resolve('/(app)/tickets')}>Ver mis tickets</Button>
		</EmptyState>
	</Card.Root>
{:else if !ticket}
	<div class="grid gap-4" aria-busy="true">
		<Skeleton class="h-4 w-40" />
		<Skeleton class="h-9 w-2/3" />
		<div class="grid gap-6 lg:grid-cols-[1fr_20rem]">
			<Skeleton class="h-72" />
			<Skeleton class="h-72" />
		</div>
	</div>
{:else}
	<!-- Encabezado del ticket -->
	<nav class="mb-3 flex items-center gap-1 text-sm text-muted-foreground" aria-label="Ruta">
		<a href={resolve('/(app)/tickets')} class="hover:text-foreground">Tickets</a>
		<ChevronRightIcon class="size-3.5" aria-hidden="true" />
		<span class="font-mono">#{ticket.id}</span>
	</nav>

	<div class="mb-6 flex flex-wrap items-start justify-between gap-4">
		<div class="min-w-0">
			<h1 class="text-2xl font-bold tracking-tight text-balance">{ticket.title}</h1>
			<div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-muted-foreground">
				<StatusBadge status={ticket.status} />
				<PriorityBadge priority={ticket.priority} class="text-foreground" />
				{#if ticket.category}
					<span class="rounded-md bg-muted px-2 py-0.5 text-xs font-medium text-foreground">
						{ticket.category}
					</span>
				{/if}
				<span>
					Abierto por <span class="text-foreground">{ticket.requester.name}</span>
					{formatRelative(ticket.created_at)}
				</span>
			</div>
		</div>

		<div class="flex gap-2">
			{#if $isStaff}
				{#if isDone}
					<Button
						variant="outline"
						disabled={saving}
						onclick={() => update({ status: 'open' }, 'Ticket reabierto')}
					>
						<RotateCcwIcon aria-hidden="true" />
						Reabrir
					</Button>
				{:else}
					<Button
						disabled={saving}
						onclick={() => update({ status: 'resolved' }, 'Ticket resuelto')}
					>
						<CircleCheckIcon aria-hidden="true" />
						Resolver
					</Button>
				{/if}
			{:else if !isClosed}
				<Button
					variant="outline"
					disabled={saving}
					onclick={() => update({ status: 'closed' }, 'Ticket cerrado')}
				>
					Cerrar ticket
				</Button>
			{/if}
		</div>
	</div>

	<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
		<!-- Conversación -->
		<section aria-labelledby="conversation-title" class="min-w-0">
			<h2 id="conversation-title" class="mb-3 flex items-center gap-2 text-sm font-semibold">
				<MessageSquareIcon class="size-4 text-muted-foreground" aria-hidden="true" />
				Conversación ({comments.length})
			</h2>

			<ol
				class="relative grid gap-4 before:absolute before:top-4 before:bottom-4 before:left-5 before:w-px before:bg-border"
			>
				<!-- La descripción es el primer mensaje del solicitante. -->
				<li class="relative flex gap-3">
					<HexAvatar
						name={ticket.requester.name}
						id={ticket.requester.id}
						size="md"
						class="ring-4 ring-background"
					/>
					<div class="min-w-0 flex-1 rounded-xl border bg-card p-4 shadow-xs">
						<div class="mb-2 flex flex-wrap items-center gap-x-2 text-sm">
							<span class="font-medium">{ticket.requester.name}</span>
							<span class="text-xs text-muted-foreground">abrió el ticket</span>
							<time
								class="text-xs text-muted-foreground"
								datetime={ticket.created_at}
								title={formatDateTime(ticket.created_at)}
							>
								{formatRelative(ticket.created_at)}
							</time>
						</div>
						{#if ticket.description}
							<p class="whitespace-pre-wrap">{ticket.description}</p>
						{:else}
							<p class="text-muted-foreground italic">Sin descripción.</p>
						{/if}
					</div>
				</li>

				{#each comments as comment (comment.id)}
					<li class="relative flex gap-3">
						<HexAvatar
							name={comment.author.name}
							id={comment.author.id}
							size="md"
							class="ring-4 ring-background"
						/>
						<div
							class={cn(
								'min-w-0 flex-1 rounded-xl border p-4 shadow-xs',
								comment.internal
									? 'border-dashed border-amber-400 bg-amber-50 dark:border-amber-500/50 dark:bg-amber-500/10'
									: 'bg-card'
							)}
						>
							<div class="mb-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
								<span class="font-medium">{comment.author.name}</span>
								<span
									class="rounded bg-muted px-1.5 py-0.5 text-[0.68rem] font-medium text-muted-foreground"
								>
									{authorRole(comment.author.id)}
								</span>
								<time
									class="text-xs text-muted-foreground"
									datetime={comment.created_at}
									title={formatDateTime(comment.created_at)}
								>
									{formatRelative(comment.created_at)}
								</time>
								{#if comment.internal}
									<span
										class="ml-auto inline-flex items-center gap-1 text-xs font-medium text-amber-700 dark:text-amber-400"
									>
										<LockIcon class="size-3" aria-hidden="true" />
										Nota interna
									</span>
								{/if}
							</div>
							<p class="whitespace-pre-wrap">{comment.body}</p>
						</div>
					</li>
				{/each}
			</ol>

			<!-- Editor de respuesta -->
			{#if isClosed}
				<p
					class="mt-6 rounded-xl border border-dashed p-4 text-center text-sm text-muted-foreground"
				>
					Este ticket está cerrado. Abre uno nuevo si necesitas más ayuda.
				</p>
			{:else}
				<form
					bind:this={formEl}
					method="POST"
					use:enhance
					novalidate
					class={cn(
						'mt-6 overflow-hidden rounded-xl border bg-card shadow-sm transition-colors',
						internal &&
							'border-amber-400 bg-amber-50/60 dark:border-amber-500/50 dark:bg-amber-500/5'
					)}
				>
					{#if $isStaff}
						<div class="flex border-b" role="tablist" aria-label="Tipo de respuesta">
							<button
								type="button"
								role="tab"
								aria-selected={!internal}
								onclick={() => ($form.internal = false)}
								class={cn(
									'border-b-2 px-4 py-2.5 text-sm font-medium transition-colors',
									!internal
										? 'border-primary text-foreground'
										: 'border-transparent text-muted-foreground hover:text-foreground'
								)}
							>
								Respuesta pública
							</button>
							<button
								type="button"
								role="tab"
								aria-selected={internal}
								onclick={() => ($form.internal = true)}
								class={cn(
									'inline-flex items-center gap-1.5 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors',
									internal
										? 'border-amber-500 text-amber-800 dark:text-amber-300'
										: 'border-transparent text-muted-foreground hover:text-foreground'
								)}
							>
								<LockIcon class="size-3.5" aria-hidden="true" />
								Nota interna
							</button>
						</div>
					{/if}

					{#if $message}
						<Alert.Root variant="destructive" class="m-3 w-auto">
							<Alert.Description>{$message}</Alert.Description>
						</Alert.Root>
					{/if}

					<Label for="body" class="sr-only">Responder</Label>
					<textarea
						id="body"
						name="body"
						rows={4}
						bind:value={$form.body}
						onkeydown={onComposerKeydown}
						aria-invalid={!!$errors.body?.length}
						aria-describedby={$errors.body?.length ? 'body-error' : undefined}
						placeholder={internal
							? 'Nota para el equipo: el cliente no la verá…'
							: $isStaff
								? `Responder a ${ticket.requester.name}…`
								: 'Escribe tu respuesta…'}
						class="block w-full resize-y bg-transparent px-4 py-3 text-sm outline-none placeholder:text-muted-foreground"
					></textarea>
					{#if $errors.body?.length}
						<p id="body-error" class="px-4 pb-2 text-sm text-destructive">{$errors.body[0]}</p>
					{/if}

					<div class="flex items-center justify-between gap-3 border-t bg-muted/40 px-4 py-2.5">
						<p class="hidden text-xs text-muted-foreground sm:block">
							{internal ? 'Solo lo verá el equipo de soporte.' : 'Ctrl + Enter para enviar'}
						</p>
						<Button type="submit" size="sm" disabled={$submitting} class="ml-auto">
							<SendIcon aria-hidden="true" />
							{$submitting ? 'Enviando…' : internal ? 'Añadir nota' : 'Enviar respuesta'}
						</Button>
					</div>
				</form>
			{/if}
		</section>

		<!-- Panel de propiedades -->
		<aside class="grid content-start gap-4">
			<Card.Root class="gap-4 py-5">
				<Card.Header class="px-5">
					<Card.Title class="text-sm">Propiedades</Card.Title>
				</Card.Header>
				<Card.Content class="grid gap-4 px-5 text-sm">
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
								{#each [...TICKET_PRIORITIES].reverse() as priority (priority)}
									<NativeSelectOption value={priority}
										>{priorityLabels[priority]}</NativeSelectOption
									>
								{/each}
							</NativeSelect>
						</div>
						<div class="grid gap-1.5">
							<div class="flex items-center justify-between">
								<Label for="assignee">Asignado a</Label>
								{#if !assignedToMe && $user}
									<button
										type="button"
										class="text-xs font-medium text-primary hover:underline disabled:opacity-50 dark:text-honey"
										disabled={saving}
										onclick={() => update({ assignee_id: $user.id }, 'Ticket asignado')}
									>
										Asignarme
									</button>
								{/if}
							</div>
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
						<div class="flex items-center justify-between gap-2">
							<span class="text-muted-foreground">Estado</span>
							<StatusBadge status={ticket.status} />
						</div>
						<div class="flex items-center justify-between gap-2">
							<span class="text-muted-foreground">Prioridad</span>
							<PriorityBadge priority={ticket.priority} />
						</div>
						<div class="flex items-center justify-between gap-2">
							<span class="text-muted-foreground">Asignado a</span>
							<span class="flex items-center gap-2">
								<HexAvatar name={ticket.assignee?.name} id={ticket.assignee?.id} size="xs" />
								{ticket.assignee?.name ?? 'Pendiente'}
							</span>
						</div>
					{/if}
				</Card.Content>
			</Card.Root>

			<Card.Root class="gap-3 py-5">
				<Card.Header class="px-5">
					<Card.Title class="text-sm">Solicitante</Card.Title>
				</Card.Header>
				<Card.Content class="flex-row items-center gap-3 px-5">
					<HexAvatar name={ticket.requester.name} id={ticket.requester.id} size="lg" />
					<div class="min-w-0">
						<p class="truncate font-medium">{ticket.requester.name}</p>
						<p class="text-xs text-muted-foreground">Cliente</p>
					</div>
				</Card.Content>
			</Card.Root>

			<Card.Root class="gap-3 py-5">
				<Card.Header class="px-5">
					<Card.Title class="text-sm">Detalles</Card.Title>
				</Card.Header>
				<Card.Content class="px-5">
					<dl class="grid gap-2.5 text-sm">
						<div class="flex justify-between gap-2">
							<dt class="text-muted-foreground">Ticket</dt>
							<dd class="font-mono">#{ticket.id}</dd>
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
							<dd title={formatDateTime(ticket.updated_at)}>{formatRelative(ticket.updated_at)}</dd>
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
		</aside>
	</div>
{/if}
