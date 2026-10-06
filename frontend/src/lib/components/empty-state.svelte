<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		children
	}: { title: string; description?: string; children?: Snippet } = $props();
</script>

<div class="flex flex-col items-center px-6 py-14 text-center">
	<!-- Tres celdas de panal, la del centro rellena de miel. -->
	<div class="relative mb-5 h-16 w-24" aria-hidden="true">
		<span class="hex absolute top-3 left-0 size-10 bg-muted"></span>
		<span class="hex absolute top-0 left-7 size-10 bg-honey/80"></span>
		<span class="hex absolute top-3 left-14 size-10 bg-muted"></span>
	</div>
	<p class="font-semibold">{title}</p>
	{#if description}
		<p class="mt-1 max-w-sm text-sm text-muted-foreground">{description}</p>
	{/if}
	{#if children}
		<div class="mt-5 flex flex-wrap justify-center gap-2">{@render children()}</div>
	{/if}
</div>
