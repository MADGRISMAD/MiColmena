<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import { api, type AssetInput } from '#lib/api/index.js';
	import AssetForm from '#lib/components/asset-form.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { isStaff } from '#lib/stores/auth.js';

	$effect(() => {
		if (!$isStaff) goto(resolve('/(app)/tickets'), { replace: true });
	});

	async function create(input: AssetInput) {
		const asset = await api.createAsset(input);
		toast.success(`Equipo ${asset.tag} creado`);
		await goto(resolve('/(app)/assets/[id]', { id: String(asset.id) }));
	}
</script>

<PageHeader
	eyebrow="Activos"
	title="Nuevo equipo"
	description="Da de alta un equipo para saber dónde está, quién lo usa y qué tickets ha tenido."
/>

{#if $isStaff}
	<Card.Root class="max-w-3xl">
		<Card.Content>
			<AssetForm submitLabel="Crear equipo" onsubmit={create}>
				{#snippet actions()}
					<Button variant="ghost" href={resolve('/(app)/assets')}>Cancelar</Button>
				{/snippet}
			</AssetForm>
		</Card.Content>
	</Card.Root>
{/if}
