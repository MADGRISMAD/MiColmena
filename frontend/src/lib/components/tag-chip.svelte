<script lang="ts">
	import { cn } from '#lib/utils.js';

	let { tag, class: className }: { tag: string; class?: string } = $props();
</script>

<span
	class={cn(
		'inline-flex items-center rounded-md border border-honey/40 bg-honey/10 px-1.5 py-px text-[0.7rem] font-medium text-amber-900 dark:text-amber-200',
		className
	)}
>
	#{tag}
</span>
