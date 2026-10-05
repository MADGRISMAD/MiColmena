<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import { api, TICKET_PRIORITIES } from '#lib/api/index.js';
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

	const { form, errors, message, submitting, enhance } = apiForm(
		ticketSchema,
		emptyTicket,
		async (data) => {
			const ticket = await api.createTicket(data);
			toast.success(`Ticket #${ticket.id} creado`);
			await goto(resolve('/(app)/tickets/[id]', { id: String(ticket.id) }));
		}
	);
</script>

<PageHeader
	title="Nuevo ticket"
	description="Cuéntanos qué pasa y te ayudaremos lo antes posible."
/>

<Card.Root class="max-w-2xl">
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
								<NativeSelectOption value={priority}>{priorityLabels[priority]}</NativeSelectOption>
							{/each}
						</NativeSelect>
					{/snippet}
				</FormField>

				<FormField id="category" label="Categoría (opcional)" errors={$errors.category}>
					{#snippet children({ id, invalid, describedBy })}
						<Input
							{id}
							name="category"
							placeholder="Ej.: Facturación"
							bind:value={$form.category}
							aria-invalid={invalid}
							aria-describedby={describedBy}
						/>
					{/snippet}
				</FormField>
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
