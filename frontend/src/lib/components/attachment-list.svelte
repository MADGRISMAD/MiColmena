<script lang="ts">
	import DownloadIcon from '@lucide/svelte/icons/download';
	import FileIcon from '@lucide/svelte/icons/file';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import { toast } from 'svelte-sonner';
	import { api, type Attachment } from '#lib/api/index.js';
	import { formatBytes, isImage, saveAttachment } from '#lib/files.js';

	let {
		items,
		canDelete = () => false,
		ondelete
	}: {
		items: Attachment[];
		canDelete?: (a: Attachment) => boolean;
		ondelete?: (a: Attachment) => void;
	} = $props();

	// Miniaturas de imágenes: se descargan con el token y se muestran como blob.
	let thumbs = $state<Record<number, string>>({});
	$effect(() => {
		for (const a of items) {
			if (isImage(a) && !(a.id in thumbs) && a.size < 5 * 1024 * 1024) {
				thumbs[a.id] = '';
				api
					.downloadAttachment(a.id)
					.then((blob) => (thumbs[a.id] = URL.createObjectURL(blob)))
					.catch(() => {});
			}
		}
	});
	$effect(() => () => Object.values(thumbs).forEach((url) => url && URL.revokeObjectURL(url)));

	async function download(a: Attachment) {
		try {
			await saveAttachment(a);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo descargar el archivo');
		}
	}

	async function remove(a: Attachment) {
		if (!confirm(`¿Borrar «${a.filename}»?`)) return;
		try {
			await api.deleteAttachment(a.id);
			ondelete?.(a);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo borrar el archivo');
		}
	}
</script>

{#if items.length > 0}
	<ul class="mt-3 flex flex-wrap gap-2" aria-label="Archivos adjuntos">
		{#each items as a (a.id)}
			<li
				class="group flex max-w-full items-center gap-2 rounded-lg border bg-background/70 p-1.5 pr-2 text-sm"
			>
				<button
					type="button"
					class="flex min-w-0 items-center gap-2 text-left"
					onclick={() => download(a)}
					title={`Descargar ${a.filename}`}
				>
					{#if thumbs[a.id]}
						<img src={thumbs[a.id]} alt="" class="size-9 rounded object-cover" />
					{:else}
						<span class="grid size-9 shrink-0 place-items-center rounded bg-muted">
							<FileIcon class="size-4 text-muted-foreground" aria-hidden="true" />
						</span>
					{/if}
					<span class="min-w-0">
						<span class="block max-w-48 truncate font-medium">{a.filename}</span>
						<span class="block text-xs text-muted-foreground">{formatBytes(a.size)}</span>
					</span>
					<DownloadIcon class="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
				</button>
				{#if canDelete(a)}
					<button
						type="button"
						class="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
						onclick={() => remove(a)}
						aria-label={`Borrar ${a.filename}`}
					>
						<Trash2Icon class="size-3.5" aria-hidden="true" />
					</button>
				{/if}
			</li>
		{/each}
	</ul>
{/if}
