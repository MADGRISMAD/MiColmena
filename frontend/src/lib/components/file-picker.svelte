<script lang="ts">
	import PaperclipIcon from '@lucide/svelte/icons/paperclip';
	import XIcon from '@lucide/svelte/icons/x';
	import { toast } from 'svelte-sonner';
	import { formatBytes, MAX_UPLOAD_BYTES } from '#lib/files.js';

	let { files = $bindable([]), disabled = false }: { files?: File[]; disabled?: boolean } =
		$props();

	let input = $state<HTMLInputElement | null>(null);

	function add(list: FileList | null) {
		for (const file of list ?? []) {
			if (file.size > MAX_UPLOAD_BYTES) {
				toast.error(`«${file.name}» supera los ${formatBytes(MAX_UPLOAD_BYTES)}`);
				continue;
			}
			if (files.length >= 10) {
				toast.error('Como máximo 10 archivos a la vez');
				break;
			}
			files = [...files, file];
		}
		if (input) input.value = '';
	}
</script>

<input
	bind:this={input}
	type="file"
	multiple
	class="hidden"
	onchange={(e) => add(e.currentTarget.files)}
	aria-hidden="true"
	tabindex="-1"
/>
<button
	type="button"
	class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-muted-foreground hover:bg-accent hover:text-foreground disabled:opacity-50"
	onclick={() => input?.click()}
	{disabled}
>
	<PaperclipIcon class="size-3.5" aria-hidden="true" />
	Adjuntar
</button>

{#if files.length > 0}
	<ul class="flex w-full flex-wrap gap-1.5" aria-label="Archivos para adjuntar">
		{#each files as file, i (file.name + i)}
			<li
				class="inline-flex items-center gap-1 rounded-md border bg-background px-2 py-0.5 text-xs"
			>
				<span class="max-w-40 truncate">{file.name}</span>
				<span class="text-muted-foreground">{formatBytes(file.size)}</span>
				<button
					type="button"
					class="rounded p-0.5 hover:bg-muted"
					onclick={() => (files = files.filter((_, j) => j !== i))}
					aria-label={`Quitar ${file.name}`}
				>
					<XIcon class="size-3" aria-hidden="true" />
				</button>
			</li>
		{/each}
	</ul>
{/if}
