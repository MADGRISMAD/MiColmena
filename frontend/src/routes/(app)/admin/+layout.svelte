<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { user } from '#lib/stores/auth.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	// La API ya protege estas rutas; aquí solo se evita mostrar una página que fallaría.
	$effect(() => {
		if ($user && $user.role !== 'admin') goto(resolve('/(app)/tickets'), { replace: true });
	});
</script>

{#if $user?.role === 'admin'}
	{@render children()}
{/if}
