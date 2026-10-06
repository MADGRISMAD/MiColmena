<script lang="ts" module>
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up';
	import ChevronsUpIcon from '@lucide/svelte/icons/chevrons-up';
	import EqualIcon from '@lucide/svelte/icons/equal';
	import type { TicketPriority } from '#lib/api/types.js';

	/** Icono y color por prioridad: se reconoce de un vistazo en listas largas. */
	export const priorityStyles: Record<
		TicketPriority,
		{ icon: typeof EqualIcon; color: string; bar: string }
	> = {
		urgent: { icon: ChevronsUpIcon, color: 'text-red-600 dark:text-red-400', bar: 'bg-red-500' },
		high: {
			icon: ChevronUpIcon,
			color: 'text-orange-600 dark:text-orange-400',
			bar: 'bg-orange-500'
		},
		medium: { icon: EqualIcon, color: 'text-amber-600 dark:text-amber-400', bar: 'bg-amber-400' },
		low: { icon: ChevronDownIcon, color: 'text-sky-600 dark:text-sky-400', bar: 'bg-sky-400' }
	};
</script>

<script lang="ts">
	import { priorityLabels } from '#lib/format.js';
	import { cn } from '#lib/utils.js';

	let {
		priority,
		compact = false,
		class: className
	}: { priority: TicketPriority; compact?: boolean; class?: string } = $props();

	const style = $derived(priorityStyles[priority]);
</script>

<span
	class={cn('inline-flex items-center gap-1 text-sm whitespace-nowrap', className)}
	title={compact ? `Prioridad ${priorityLabels[priority].toLowerCase()}` : undefined}
>
	<style.icon class={cn('size-4 shrink-0', style.color)} aria-hidden="true" />
	{#if compact}
		<span class="sr-only">Prioridad {priorityLabels[priority].toLowerCase()}</span>
	{:else}
		{priorityLabels[priority]}
	{/if}
</span>
