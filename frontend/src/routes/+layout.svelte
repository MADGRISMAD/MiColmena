<script lang="ts">
	import './layout.css';
	import { ModeWatcher } from 'mode-watcher';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	<title>MiColmena</title>
	<link rel="icon" type="image/png" href="/favicon.png" />
</svelte:head>

<ModeWatcher />

{@render children()}
