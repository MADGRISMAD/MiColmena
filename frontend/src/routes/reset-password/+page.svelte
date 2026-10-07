<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { api, ApiError } from '#lib/api/index.js';
	import AuthCard from '#lib/components/auth-card.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { homeFor } from '#lib/navigation.js';
	import { setSession } from '#lib/stores/auth.js';

	const token = page.url.searchParams.get('token') ?? '';
	let password = $state('');
	let confirm = $state('');
	let errors = $state<Record<string, string>>({});
	let saving = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		errors = {};
		if (password.length < 8) errors.password = 'Debe tener al menos 8 caracteres';
		else if (password !== confirm) errors.confirm = 'Las contraseñas no coinciden';
		if (Object.keys(errors).length) return;
		saving = true;
		try {
			const res = await api.resetPassword(token, password);
			setSession(res.token, res.user);
			await goto(resolve(homeFor(res.user)), { replace: true });
		} catch (err) {
			errors =
				err instanceof ApiError && Object.keys(err.fields).length
					? err.fields
					: { token: err instanceof Error ? err.message : 'No se pudo cambiar' };
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head><title>Nueva contraseña · BeHIve</title></svelte:head>

<AuthCard title="Elige una contraseña nueva" description="Después entrarás directamente.">
	{#if !token || errors.token}
		<Alert.Root variant="destructive" class="mb-4">
			<Alert.Description>
				{errors.token ?? 'Falta el enlace de recuperación.'}
				<a href={resolve('/forgot-password')} class="underline">Pide uno nuevo</a>.
			</Alert.Description>
		</Alert.Root>
	{/if}
	{#if token}
		<form class="grid gap-4" onsubmit={submit} novalidate>
			<div class="grid gap-1.5">
				<Label for="password">Nueva contraseña</Label>
				<Input
					id="password"
					type="password"
					autocomplete="new-password"
					bind:value={password}
					aria-invalid={!!errors.password}
				/>
				{#if errors.password}<p class="text-sm text-destructive">{errors.password}</p>{/if}
			</div>
			<div class="grid gap-1.5">
				<Label for="confirm">Repite la contraseña</Label>
				<Input
					id="confirm"
					type="password"
					autocomplete="new-password"
					bind:value={confirm}
					aria-invalid={!!errors.confirm}
				/>
				{#if errors.confirm}<p class="text-sm text-destructive">{errors.confirm}</p>{/if}
			</div>
			<Button type="submit" disabled={saving} class="w-full">
				{saving ? 'Guardando…' : 'Guardar y entrar'}
			</Button>
		</form>
	{/if}

	{#snippet footer()}
		<a href={resolve('/login')} class="text-primary underline dark:text-honey"
			>Volver a iniciar sesión</a
		>
	{/snippet}
</AuthCard>
