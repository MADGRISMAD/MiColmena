<script lang="ts" module>
	// Colores cálidos, todos con contraste suficiente para las iniciales.
	const palette = [
		'bg-amber-200 text-amber-900 dark:bg-amber-400/25 dark:text-amber-200',
		'bg-orange-200 text-orange-900 dark:bg-orange-400/25 dark:text-orange-200',
		'bg-rose-200 text-rose-900 dark:bg-rose-400/25 dark:text-rose-200',
		'bg-emerald-200 text-emerald-900 dark:bg-emerald-400/25 dark:text-emerald-200',
		'bg-sky-200 text-sky-900 dark:bg-sky-400/25 dark:text-sky-200',
		'bg-violet-200 text-violet-900 dark:bg-violet-400/25 dark:text-violet-200',
		'bg-teal-200 text-teal-900 dark:bg-teal-400/25 dark:text-teal-200',
		'bg-lime-200 text-lime-900 dark:bg-lime-400/25 dark:text-lime-200'
	];

	export function initials(name: string): string {
		const words = name.trim().split(/\s+/).filter(Boolean);
		const letters = words.length > 1 ? words[0][0] + words[1][0] : (words[0] ?? '?').slice(0, 2);
		return letters.toUpperCase();
	}

	export function colorFor(seed: string | number): string {
		const text = String(seed);
		let hash = 0;
		for (let i = 0; i < text.length; i++) hash = (hash * 31 + text.charCodeAt(i)) >>> 0;
		return palette[hash % palette.length];
	}
</script>

<script lang="ts">
	import UserRoundXIcon from '@lucide/svelte/icons/user-round-x';
	import { cn } from '#lib/utils.js';

	let {
		name,
		id,
		size = 'sm',
		class: className
	}: {
		/** Sin nombre se muestra el hexágono vacío de "sin asignar". */
		name?: string | null;
		id?: number | string;
		size?: 'xs' | 'sm' | 'md' | 'lg';
		class?: string;
	} = $props();

	const sizes = {
		xs: 'size-6 text-[0.6rem]',
		sm: 'size-8 text-[0.7rem]',
		md: 'size-10 text-xs',
		lg: 'size-12 text-sm'
	};
</script>

{#if name}
	<span
		class={cn(
			'hex inline-flex shrink-0 items-center justify-center font-semibold select-none',
			sizes[size],
			colorFor(id ?? name),
			className
		)}
		title={name}
		aria-hidden="true"
	>
		{initials(name)}
	</span>
{:else}
	<span
		class={cn(
			'hex inline-flex shrink-0 items-center justify-center bg-muted text-muted-foreground',
			sizes[size],
			className
		)}
		aria-hidden="true"
	>
		<UserRoundXIcon class="size-[45%]" />
	</span>
{/if}
