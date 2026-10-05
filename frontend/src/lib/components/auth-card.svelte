<script lang="ts">
	import type { Snippet } from 'svelte';
	import { resolve } from '$app/paths';
	import * as Card from '#lib/components/ui/card/index.js';

	let {
		title,
		description,
		children,
		footer
	}: { title: string; description: string; children: Snippet; footer: Snippet } = $props();
</script>

<div class="flex min-h-screen flex-col items-center justify-center gap-6 bg-muted/40 p-4">
	<a href={resolve('/')} class="flex items-center gap-2 text-lg font-semibold">
		<img src="/logo.png" alt="" class="size-9" />
		MiColmena
	</a>
	<Card.Root class="w-full max-w-sm">
		<Card.Header>
			<Card.Title class="text-xl">{title}</Card.Title>
			<Card.Description>{description}</Card.Description>
		</Card.Header>
		<Card.Content>
			{@render children()}
		</Card.Content>
		<Card.Footer class="justify-center text-sm text-muted-foreground">
			{@render footer()}
		</Card.Footer>
	</Card.Root>
</div>
