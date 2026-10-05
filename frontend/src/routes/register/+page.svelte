<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { api } from '#lib/api/index.js';
	import AuthCard from '#lib/components/auth-card.svelte';
	import FormField from '#lib/components/form-field.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { apiForm } from '#lib/forms.js';
	import { emptyRegister, registerSchema } from '#lib/schemas.js';
	import { setSession } from '#lib/stores/auth.js';

	const { form, errors, message, submitting, enhance } = apiForm(
		registerSchema,
		emptyRegister,
		async (data) => {
			const res = await api.register(data.name, data.email, data.password);
			setSession(res.token, res.user);
			await goto(resolve('/(app)/tickets'), { replace: true });
		}
	);
</script>

<svelte:head><title>Crear cuenta · MiColmena</title></svelte:head>

<AuthCard title="Crear cuenta" description="Regístrate para abrir y seguir tus tickets.">
	<form method="POST" use:enhance class="grid gap-4" novalidate>
		{#if $message}
			<Alert.Root variant="destructive">
				<Alert.Description>{$message}</Alert.Description>
			</Alert.Root>
		{/if}

		<FormField id="name" label="Nombre" errors={$errors.name}>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					name="name"
					autocomplete="name"
					bind:value={$form.name}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>

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

		<FormField
			id="password"
			label="Contraseña"
			errors={$errors.password}
			description="Mínimo 8 caracteres."
		>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					name="password"
					type="password"
					autocomplete="new-password"
					bind:value={$form.password}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>

		<FormField id="confirm" label="Repite la contraseña" errors={$errors.confirm}>
			{#snippet children({ id, invalid, describedBy })}
				<Input
					{id}
					name="confirm"
					type="password"
					autocomplete="new-password"
					bind:value={$form.confirm}
					aria-invalid={invalid}
					aria-describedby={describedBy}
				/>
			{/snippet}
		</FormField>

		<Button type="submit" disabled={$submitting} class="w-full">
			{$submitting ? 'Creando cuenta…' : 'Crear cuenta'}
		</Button>
	</form>

	{#snippet footer()}
		¿Ya tienes cuenta?&nbsp;<a href={resolve('/login')} class="text-primary underline">
			Inicia sesión
		</a>
	{/snippet}
</AuthCard>
