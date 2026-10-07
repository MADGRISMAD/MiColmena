<script lang="ts">
	import { cn } from '#lib/utils.js';

	let {
		tone = 'light',
		class: className
	}: {
		/** "dark" para fondos oscuros (barra lateral, paneles de marca). */
		tone?: 'light' | 'dark';
		class?: string;
	} = $props();
</script>

<span class={cn('inline-flex items-center gap-2.5 font-semibold tracking-tight', className)}>
	<span class="hex grid size-9 shrink-0 place-items-center bg-honey">
		<img src="/logo.png" alt="" class="size-7" />
	</span>
	<span class={cn('text-lg', tone === 'dark' ? 'text-white' : 'text-foreground')}>
		Be<span class={tone === 'dark' ? 'text-honey' : 'text-primary dark:text-honey'}>HI</span>ve
	</span>
</span>
