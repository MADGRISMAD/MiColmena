<script lang="ts">
	import type { Snippet } from 'svelte';

	let { title, description, actions }: { title: string; description?: string; actions?: Snippet } =
		$props();
</script>

<svelte:head><title>{title} · MiColmena</title></svelte:head>

<div class="mb-6 flex flex-wrap items-end justify-between gap-4">
	<div>
		<h1 class="text-2xl font-bold tracking-tight">{title}</h1>
		{#if description}
			<p class="mt-1 text-muted-foreground">{description}</p>
		{/if}
	</div>
	{#if actions}
		<div class="flex gap-2">{@render actions()}</div>
	{/if}
</div>
