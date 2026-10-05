<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Label } from '#lib/components/ui/label/index.js';

	let {
		id,
		label,
		errors,
		description,
		children
	}: {
		id: string;
		label: string;
		errors?: string[];
		description?: string;
		children: Snippet<[{ id: string; invalid: boolean; describedBy: string | undefined }]>;
	} = $props();

	const invalid = $derived(!!errors?.length);
	const describedBy = $derived(
		invalid ? `${id}-error` : description ? `${id}-description` : undefined
	);
</script>

<div class="grid gap-1.5">
	<Label for={id}>{label}</Label>
	{@render children({ id, invalid, describedBy })}
	{#if invalid}
		<p id="{id}-error" class="text-sm text-destructive">{errors?.[0]}</p>
	{:else if description}
		<p id="{id}-description" class="text-sm text-muted-foreground">{description}</p>
	{/if}
</div>
