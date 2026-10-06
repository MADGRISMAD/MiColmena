<script lang="ts">
	import XIcon from '@lucide/svelte/icons/x';

	let {
		tags,
		suggestions = [],
		disabled = false,
		onchange
	}: {
		tags: string[];
		suggestions?: string[];
		disabled?: boolean;
		onchange: (tags: string[]) => void;
	} = $props();

	let draft = $state('');
	const listId = `tags-${Math.random().toString(36).slice(2)}`;

	function normalize(tag: string) {
		return tag.trim().toLowerCase().split(/\s+/).filter(Boolean).join('-');
	}

	function commit() {
		const tag = normalize(draft);
		draft = '';
		if (tag && !tags.includes(tag)) onchange([...tags, tag]);
	}
</script>

<div
	class="flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border border-input bg-transparent px-2 py-1.5 shadow-xs focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50"
>
	{#each tags as tag (tag)}
		<span
			class="inline-flex items-center gap-0.5 rounded bg-honey/15 py-0.5 pr-0.5 pl-1.5 text-xs font-medium"
		>
			#{tag}
			<button
				type="button"
				class="rounded p-0.5 hover:bg-honey/30 disabled:opacity-50"
				{disabled}
				onclick={() => onchange(tags.filter((t) => t !== tag))}
				aria-label={`Quitar la etiqueta ${tag}`}
			>
				<XIcon class="size-3" aria-hidden="true" />
			</button>
		</span>
	{/each}
	<input
		id="tags"
		bind:value={draft}
		list={listId}
		{disabled}
		placeholder={tags.length ? '' : 'Añadir etiqueta…'}
		class="min-w-20 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
		onkeydown={(e) => {
			if (e.key === 'Enter' || e.key === ',') {
				e.preventDefault();
				commit();
			} else if (e.key === 'Backspace' && !draft && tags.length) {
				onchange(tags.slice(0, -1));
			}
		}}
		onblur={commit}
	/>
	<datalist id={listId}>
		{#each suggestions.filter((s) => !tags.includes(s)) as s (s)}
			<option value={s}></option>
		{/each}
	</datalist>
</div>
