<script lang="ts">
	import ThumbsDownIcon from '@lucide/svelte/icons/thumbs-down';
	import ThumbsUpIcon from '@lucide/svelte/icons/thumbs-up';
	import { toast } from 'svelte-sonner';
	import { api, type Ticket } from '#lib/api/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { cn } from '#lib/utils.js';

	let { ticket, onrated }: { ticket: Ticket; onrated: (t: Ticket) => void } = $props();

	let rating = $state<'good' | 'bad' | ''>('');
	let comment = $state('');
	let sending = $state(false);
	let editing = $state(false);

	$effect(() => {
		rating = ticket.satisfaction;
		comment = ticket.satisfaction_comment;
	});

	async function send() {
		if (!rating) return;
		sending = true;
		try {
			onrated(await api.rateTicket(ticket.id, rating, comment));
			editing = false;
			toast.success('¡Gracias por tu valoración!');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo guardar la valoración');
		} finally {
			sending = false;
		}
	}
</script>

<section
	class="bg-honeycomb rounded-xl border border-honey/50 bg-honey/5 p-5"
	aria-labelledby="csat-title"
>
	{#if ticket.satisfaction && !editing}
		<p id="csat-title" class="font-semibold">Gracias por tu valoración</p>
		<p class="mt-1 text-sm text-muted-foreground">
			Dijiste que la atención fue
			<strong class="text-foreground">{ticket.satisfaction === 'good' ? 'buena' : 'mala'}</strong>.
			<button
				type="button"
				class="text-primary underline dark:text-honey"
				onclick={() => (editing = true)}>Cambiar</button
			>
		</p>
	{:else}
		<p id="csat-title" class="font-semibold">¿Cómo te atendimos?</p>
		<p class="mt-1 text-sm text-muted-foreground">Tu opinión ayuda al equipo a mejorar.</p>
		<div class="mt-4 flex gap-2" role="group" aria-label="Valoración">
			{#each [{ value: 'good' as const, label: 'Bien', icon: ThumbsUpIcon }, { value: 'bad' as const, label: 'Mal', icon: ThumbsDownIcon }] as option (option.value)}
				<button
					type="button"
					aria-pressed={rating === option.value}
					onclick={() => (rating = option.value)}
					class={cn(
						'inline-flex flex-1 items-center justify-center gap-2 rounded-lg border bg-card px-4 py-2.5 text-sm font-medium transition-colors hover:bg-muted',
						rating === option.value &&
							(option.value === 'good'
								? 'border-emerald-500 bg-emerald-50 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300'
								: 'border-red-500 bg-red-50 text-red-800 dark:bg-red-500/15 dark:text-red-300')
					)}
				>
					<option.icon class="size-4" aria-hidden="true" />
					{option.label}
				</button>
			{/each}
		</div>
		{#if rating}
			<label for="csat-comment" class="mt-4 block text-sm font-medium">
				¿Algo que quieras contarnos? <span class="font-normal text-muted-foreground"
					>(opcional)</span
				>
			</label>
			<textarea
				id="csat-comment"
				bind:value={comment}
				rows={2}
				maxlength={2000}
				class="mt-1.5 block w-full rounded-md border bg-card px-3 py-2 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
			></textarea>
			<Button class="mt-3" size="sm" disabled={sending} onclick={send}>
				{sending ? 'Enviando…' : 'Enviar valoración'}
			</Button>
		{/if}
	{/if}
</section>
