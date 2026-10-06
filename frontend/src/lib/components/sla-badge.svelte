<script lang="ts">
	import SirenIcon from '@lucide/svelte/icons/siren';
	import TimerIcon from '@lucide/svelte/icons/timer';
	import type { Ticket } from '#lib/api/types.js';
	import { formatDateTime } from '#lib/format.js';
	import { formatDuration, slaState } from '#lib/sla.js';
	import { cn } from '#lib/utils.js';

	let {
		ticket,
		always = false,
		class: className
	}: {
		ticket: Ticket;
		/** Mostrar también cuando va a tiempo (en la lista solo se muestran riesgos y vencidos). */
		always?: boolean;
		class?: string;
	} = $props();

	const state = $derived(slaState(ticket));
	const label = $derived(state?.target === 'first_response' ? 'Primera respuesta' : 'Resolución');
</script>

{#if state && (always || state.breached || state.atRisk)}
	<span
		class={cn(
			'inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-medium whitespace-nowrap',
			state.breached
				? 'bg-red-100 text-red-800 dark:bg-red-500/15 dark:text-red-300'
				: state.atRisk
					? 'bg-amber-100 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300'
					: 'bg-muted text-muted-foreground',
			className
		)}
		title={`${label}: ${formatDateTime(state.due.toISOString())}`}
	>
		{#if state.breached}
			<SirenIcon class="size-3" aria-hidden="true" />
			Vencido hace {formatDuration(state.remaining)}
		{:else}
			<TimerIcon class="size-3" aria-hidden="true" />
			Vence en {formatDuration(state.remaining)}
		{/if}
	</span>
{/if}
