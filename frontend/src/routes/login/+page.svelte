<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { api } from '#lib/api/index.js';
	import AuthCard from '#lib/components/auth-card.svelte';
	import FormField from '#lib/components/form-field.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { apiForm } from '#lib/forms.js';
	import { homeFor, safeRedirect } from '#lib/navigation.js';
	import { emptyLogin, loginSchema } from '#lib/schemas.js';
	import { setSession } from '#lib/stores/auth.js';

	const { form, errors, message, submitting, enhance } = apiForm(
		loginSchema,
		emptyLogin,
		async (data) => {
			const res = await api.login(data.email, data.password);
			setSession(res.token, res.user);
			const target = safeRedirect(
				page.url.searchParams.get('redirect'),
				resolve(homeFor(res.user))
			);
			await goto(target, { replace: true });
		}
	);
</script>

<svelte:head><title>Iniciar sesión · MiColmena</title></svelte:head>

<AuthCard title="Iniciar sesión" description="Entra con tu email y contraseña.">
	<form method="POST" use:enhance class="grid gap-4" novalidate>
		{#if $message}
			<Alert.Root variant="destructive">
				<Alert.Description>{$message}</Alert.Description>
			</Alert.Root>
		{/if}

		<FormField id="email" label="Email" errors={$errors.email}>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					name="email"
					type="email"
					autocomplete="email"
					bind:value={$form.email}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>

		<FormField id="password" label="Contraseña" errors={$errors.password}>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					name="password"
					type="password"
					autocomplete="current-password"
					bind:value={$form.password}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>

		<a
			href={resolve('/forgot-password')}
			class="-mt-2 justify-self-end text-sm text-muted-foreground underline-offset-4 hover:underline"
		>
			¿Olvidaste tu contraseña?
		</a>

		<Button type="submit" disabled={$submitting} class="w-full">
			{$submitting ? 'Entrando…' : 'Entrar'}
		</Button>
	</form>

	{#snippet footer()}
		¿No tienes cuenta?&nbsp;<a
			href={resolve('/register')}
			class="text-primary underline dark:text-honey"
		>
			Regístrate
		</a>
	{/snippet}
</AuthCard>
