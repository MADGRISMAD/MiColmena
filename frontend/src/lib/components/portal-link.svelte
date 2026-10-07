<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import { Button } from '#lib/components/ui/button/index.js';

	let { slug }: { slug: string } = $props();

	const url = $derived(`${location.origin}/e/${slug}`);
	let copied = $state(false);

	async function copy() {
		try {
			await navigator.clipboard.writeText(url);
			copied = true;
			setTimeout(() => (copied = false), 2000);
		} catch {
			// Sin permiso para el portapapeles: el enlace se puede seleccionar a mano.
		}
	}
</script>

<div class="flex flex-wrap items-center gap-2">
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- enlace absoluto para compartir -->
	<a
		href={url}
		target="_blank"
		rel="noopener"
		class="min-w-0 flex-1 truncate rounded-md border bg-muted/50 px-3 py-2 font-mono text-sm"
		data-testid="portal-url">{url}</a
	>
	<Button variant="outline" size="sm" onclick={copy}>
		{#if copied}
			<CheckIcon aria-hidden="true" /> Copiado
		{:else}
			<CopyIcon aria-hidden="true" /> Copiar
		{/if}
	</Button>
</div>
