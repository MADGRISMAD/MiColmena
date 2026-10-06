<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import { api, ApiError, type Article, type ArticleInput } from '#lib/api/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { isStaff, user } from '#lib/stores/auth.js';

	let { article }: { article?: Article } = $props();

	let draft = $state<ArticleInput>({ title: '', body: '', category: '', published: true });
	let errors = $state<Record<string, string>>({});
	let saving = $state(false);

	$effect(() => {
		if (article) {
			draft = {
				title: article.title,
				body: article.body,
				category: article.category,
				published: article.published
			};
		}
	});

	async function save(event: SubmitEvent) {
		event.preventDefault();
		errors = {};
		saving = true;
		try {
			const saved = article
				? await api.updateArticle(article.id, draft)
				: await api.createArticle(draft);
			toast.success(draft.published ? 'Artículo publicado' : 'Borrador guardado');
			await goto(resolve('/help/[id]', { id: String(saved.id) }));
		} catch (err) {
			if (err instanceof ApiError && Object.keys(err.fields).length) errors = err.fields;
			else toast.error(err instanceof Error ? err.message : 'No se pudo guardar');
		} finally {
			saving = false;
		}
	}

	async function remove() {
		if (!article || !confirm('¿Borrar este artículo?')) return;
		await api.deleteArticle(article.id);
		toast.success('Artículo borrado');
		await goto(resolve('/help'));
	}
</script>

{#if $user && !$isStaff}
	<p class="text-muted-foreground">Solo el equipo de soporte puede editar la ayuda.</p>
{:else}
	<form class="grid max-w-3xl gap-5" onsubmit={save} novalidate>
		<h1 class="text-2xl font-bold tracking-tight">
			{article ? 'Editar artículo' : 'Nuevo artículo'}
		</h1>
		<div class="grid gap-1.5">
			<Label for="title">Título</Label>
			<Input id="title" bind:value={draft.title} aria-invalid={!!errors.title} />
			{#if errors.title}<p class="text-sm text-destructive">{errors.title}</p>{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="category">Sección</Label>
			<Input id="category" bind:value={draft.category} placeholder="Por ejemplo: Facturación" />
		</div>
		<div class="grid gap-1.5">
			<Label for="body">Contenido</Label>
			<Textarea id="body" bind:value={draft.body} rows={16} aria-invalid={!!errors.body} />
			{#if errors.body}<p class="text-sm text-destructive">{errors.body}</p>{/if}
		</div>
		<label class="flex items-center gap-2 text-sm">
			<input type="checkbox" class="size-4 accent-amber-500" bind:checked={draft.published} />
			Publicado (visible para todos, sin iniciar sesión)
		</label>
		<div class="flex flex-wrap gap-2">
			<Button type="submit" disabled={saving}>{saving ? 'Guardando…' : 'Guardar'}</Button>
			<Button variant="ghost" href={resolve('/help')}>Cancelar</Button>
			{#if article}
				<Button type="button" variant="destructive" class="ml-auto" onclick={remove}>Borrar</Button>
			{/if}
		</div>
	</form>
{/if}
