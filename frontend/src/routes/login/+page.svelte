<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import BuildingIcon from '@lucide/svelte/icons/building-2';
	import { api, ApiError, type AuthResponse, type OrgChoice } from '#lib/api/index.js';
	import AuthCard from '#lib/components/auth-card.svelte';
	import FormField from '#lib/components/form-field.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { apiForm } from '#lib/forms.js';
	import { homeFor, safeRedirect } from '#lib/navigation.js';
	import { emptyLogin, loginSchema } from '#lib/schemas.js';
	import { setSession, takeLogoutReason } from '#lib/stores/auth.js';

	// Desde el portal de una empresa se entra directo a esa empresa (?org=slug).
	const portal = page.url.searchParams.get('org') ?? '';
	let portalName = $state('');
	$effect(() => {
		if (portal)
			api
				.getPortal(portal)
				.then((p) => (portalName = p.name))
				.catch(() => {});
	});

	// Por qué se cerró la sesión anterior (otro dispositivo, contraseña cambiada…).
	const reason = takeLogoutReason();

	/** Si el email tiene cuenta en varias empresas, se elige una. */
	let choices = $state<OrgChoice[]>([]);
	let credentials = { email: '', password: '' };
	let choosing = $state(false);

	async function finish(res: AuthResponse) {
		setSession(res.token, res.user);
		const target = safeRedirect(page.url.searchParams.get('redirect'), resolve(homeFor(res.user)));
		await goto(target, { replace: true });
	}

	const { form, errors, message, submitting, enhance } = apiForm(
		loginSchema,
		emptyLogin,
		async (data) => {
			try {
				await finish(await api.login(data.email, data.password, portal));
			} catch (err) {
				if (err instanceof ApiError && err.status === 409) {
					credentials = { email: data.email, password: data.password };
					choices = (err.data?.organizations as OrgChoice[]) ?? [];
					return;
				}
				throw err;
			}
		}
	);

	let chooseError = $state('');
	async function choose(org: OrgChoice) {
		choosing = true;
		chooseError = '';
		try {
			await finish(await api.login(credentials.email, credentials.password, org.slug));
		} catch (err) {
			chooseError = err instanceof Error ? err.message : 'No se pudo iniciar sesión';
		} finally {
			choosing = false;
		}
	}
</script>

<svelte:head><title>Iniciar sesión · MiColmena</title></svelte:head>

<AuthCard
	title={choices.length ? '¿A qué empresa quieres entrar?' : 'Iniciar sesión'}
	description={choices.length
		? 'Tienes cuenta en varias empresas con este email.'
		: portalName
			? `Entra al soporte de ${portalName}.`
			: 'Entra con tu email y contraseña.'}
	headline={portalName ? `Soporte de ${portalName}` : undefined}
>
	{#if choices.length}
		<ul class="grid gap-2" aria-label="Empresas">
			{#each choices as org (org.slug)}
				<li>
					<button
						type="button"
						class="flex w-full items-center gap-3 rounded-lg border bg-card p-3 text-left hover:border-honey hover:bg-honey/5 disabled:opacity-50"
						disabled={choosing}
						onclick={() => choose(org)}
					>
						<span class="hex grid size-9 shrink-0 place-items-center bg-honey/20 text-amber-800">
							<BuildingIcon class="size-4" aria-hidden="true" />
						</span>
						<span class="font-medium">{org.name}</span>
					</button>
				</li>
			{/each}
		</ul>
		{#if chooseError}<p class="mt-3 text-sm text-destructive">{chooseError}</p>{/if}
		<button
			type="button"
			class="mt-4 text-sm text-muted-foreground underline"
			onclick={() => (choices = [])}>Usar otro email</button
		>
	{:else}
		<form method="POST" use:enhance class="grid gap-4" novalidate>
			{#if reason && !$message}
				<Alert.Root>
					<Alert.Description>{reason}</Alert.Description>
				</Alert.Root>
			{/if}
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
	{/if}

	{#snippet footer()}
		{#if portal}
			¿Primera vez?&nbsp;<a
				href={resolve(`/register?org=${encodeURIComponent(portal)}`)}
				class="text-primary underline dark:text-honey">Crea tu cuenta</a
			>
		{:else}
			¿Tu empresa aún no usa MiColmena?&nbsp;<a
				href={resolve('/signup')}
				class="text-primary underline dark:text-honey">Crea una cuenta gratis</a
			>
		{/if}
	{/snippet}
</AuthCard>
