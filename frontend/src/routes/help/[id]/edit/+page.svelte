<script lang="ts">
	import { page } from '$app/state';
	import { api, type Article } from '#lib/api/index.js';
	import ArticleEditor from '#lib/components/article-editor.svelte';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';

	let article = $state<Article | null>(null);
	let missing = $state(false);
	$effect(() => {
		api
			.getArticle(Number(page.params.id))
			.then((a) => (article = a))
			.catch(() => (missing = true));
	});
</script>

<svelte:head><title>Editar artículo · BeHIve</title></svelte:head>

{#if missing}
	<p class="text-muted-foreground">El artículo no existe.</p>
{:else if article}
	<ArticleEditor {article} />
{:else}
	<Skeleton class="h-96" />
{/if}
