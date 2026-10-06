<script lang="ts">
	import { resolve } from '$app/paths';
	import { api } from '#lib/api/index.js';
	import AuthCard from '#lib/components/auth-card.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';

	let email = $state('');
	let sent = $state(false);
	let sending = $state(false);
	let error = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		if (!/^\S+@\S+\.\S+$/.test(email.trim())) {
			error = 'Introduce un email válido';
			return;
		}
		sending = true;
		try {
			await api.forgotPassword(email.trim());
			sent = true;
		} catch (err) {
			error = err instanceof Error ? err.message : 'No se pudo enviar';
		} finally {
			sending = false;
		}
	}
</script>

<svelte:head><title>Recuperar contraseña · MiColmena</title></svelte:head>

<AuthCard
	title="Recuperar contraseña"
	description="Te enviaremos un enlace para elegir una contraseña nueva."
>
	{#if sent}
		<Alert.Root>
			<Alert.Description>
				Si hay una cuenta con <strong>{email}</strong>, en unos minutos recibirás el enlace. Revisa
				también la carpeta de spam. El enlace vale durante 1 hora.
			</Alert.Description>
		</Alert.Root>
	{:else}
		<form class="grid gap-4" onsubmit={submit} novalidate>
			<div class="grid gap-1.5">
				<Label for="email">Email</Label>
				<Input
					id="email"
					type="email"
					autocomplete="email"
					bind:value={email}
					aria-invalid={!!error}
					aria-describedby={error ? 'email-error' : undefined}
				/>
				{#if error}<p id="email-error" class="text-sm text-destructive">{error}</p>{/if}
			</div>
			<Button type="submit" disabled={sending} class="w-full">
				{sending ? 'Enviando…' : 'Enviar enlace'}
			</Button>
		</form>
	{/if}

	{#snippet footer()}
		<a href={resolve('/login')} class="text-primary underline dark:text-honey"
			>Volver a iniciar sesión</a
		>
	{/snippet}
</AuthCard>
