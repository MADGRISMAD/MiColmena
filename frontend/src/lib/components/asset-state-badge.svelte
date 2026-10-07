<script lang="ts" module>
	import type { AssetState } from '#lib/api/types.js';

	/** Colores por estado del equipo (píldora y punto). */
	export const assetStateStyles: Record<AssetState, { pill: string; dot: string }> = {
		in_use: {
			pill: 'bg-emerald-50 text-emerald-800 ring-emerald-600/20 dark:bg-emerald-400/10 dark:text-emerald-300 dark:ring-emerald-400/25',
			dot: 'bg-emerald-500'
		},
		in_stock: {
			pill: 'bg-sky-50 text-sky-800 ring-sky-600/20 dark:bg-sky-400/10 dark:text-sky-300 dark:ring-sky-400/25',
			dot: 'bg-sky-500'
		},
		in_repair: {
			pill: 'bg-amber-50 text-amber-800 ring-amber-600/25 dark:bg-amber-400/10 dark:text-amber-300 dark:ring-amber-400/25',
			dot: 'bg-amber-500'
		},
		retired: {
			pill: 'bg-stone-100 text-stone-700 ring-stone-500/20 dark:bg-stone-400/10 dark:text-stone-300 dark:ring-stone-400/25',
			dot: 'bg-stone-400'
		}
	};
</script>

<script lang="ts">
	import { assetStateLabels } from '#lib/format.js';
	import { cn } from '#lib/utils.js';

	let { state, class: className }: { state: AssetState; class?: string } = $props();
</script>

<span
	class={cn(
		'inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ring-1 ring-inset',
		assetStateStyles[state].pill,
		className
	)}
>
	<span class={cn('size-1.5 rounded-full', assetStateStyles[state].dot)} aria-hidden="true"></span>
	{assetStateLabels[state]}
</span>
