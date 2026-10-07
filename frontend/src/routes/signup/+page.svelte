<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { api, ApiError, type SignupInput } from '#lib/api/index.js';
	import AuthCard from '#lib/components/auth-card.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { FREE_PEOPLE, PEOPLE_STOPS, peopleLabel } from '#lib/pricing.js';
	import { setSession } from '#lib/stores/auth.js';

	// La calculadora de la portada puede mandar el tamaño elegido (?people=100).
	const fromQuery = Number(page.url.searchParams.get('people'));
	let draft = $state<SignupInput>({
		company: '',
		people: PEOPLE_STOPS.includes(fromQuery) ? fromQuery : FREE_PEOPLE,
		name: '',
		email: '',
		password: '',
		accept_terms: false
	});
	let errors = $state<Record<string, string>>({});
	let formError = $state('');
	let sending = $state(false);

	function validate() {
		const e: Record<string, string> = {};
		if (!draft.company.trim()) e.company = 'Escribe el nombre de tu empresa';
		if (!draft.name.trim()) e.name = 'Escribe tu nombre';
		if (!/^\S+@\S+\.\S+$/.test(draft.email.trim())) e.email = 'Escribe un email válido';
		if (draft.password.length < 8) e.password = 'Debe tener al menos 8 caracteres';
		if (!draft.accept_terms) e.accept_terms = 'Debes aceptar para continuar';
		return e;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		formError = '';
		errors = validate();
		if (Object.keys(errors).length) return;
		sending = true;
		try {
			const res = await api.signup({
				...draft,
				people: Number(draft.people),
				company: draft.company.trim(),
				name: draft.name.trim(),
				email: draft.email.trim()
			});
			setSession(res.token, res.user);
			await goto(resolve('/(app)/dashboard?bienvenida=1'), { replace: true });
		} catch (err) {
			if (err instanceof ApiError && Object.keys(err.fields).length) errors = err.fields;
			else formError = err instanceof Error ? err.message : 'No se pudo crear la cuenta';
		} finally {
			sending = false;
		}
	}

	const fieldError = (name: string) => errors[name];
</script>

<svelte:head><title>Crear cuenta gratis · BeHIve</title></svelte:head>

<AuthCard
	wide
	title="Crea la cuenta de tu empresa"
	description={`Gratis para siempre con 1 agente y hasta ${FREE_PEOPLE} personas. Sin tarjeta.`}
	headline="Tu mesa de ayuda lista en minutos."
	points={[
		'Un portal propio para que tus clientes abran tickets.',
		'Todas las funciones desde el primer día.',
		'Precios en pesos cuando tu equipo crezca.'
	]}
>
	<form class="grid gap-4 sm:grid-cols-2" onsubmit={submit} novalidate>
		{#if formError}
			<Alert.Root variant="destructive" class="sm:col-span-2">
				<Alert.Description>{formError}</Alert.Description>
			</Alert.Root>
		{/if}

		<div class="grid gap-1.5 sm:col-span-2">
			<Label for="company">Nombre de tu empresa</Label>
			<Input
				id="company"
				autocomplete="organization"
				bind:value={draft.company}
				aria-invalid={!!fieldError('company')}
				aria-describedby={fieldError('company') ? 'company-error' : undefined}
			/>
			{#if fieldError('company')}
				<p id="company-error" class="text-sm text-destructive">{fieldError('company')}</p>
			{/if}
		</div>
		<div class="grid gap-1.5 sm:col-span-2">
			<Label for="people">¿Cuántas personas trabajan en tu empresa?</Label>
			<NativeSelect id="people" class="w-full" bind:value={draft.people}>
				{#each PEOPLE_STOPS as stop (stop)}
					<NativeSelectOption value={stop}>{peopleLabel(stop)} personas</NativeSelectOption>
				{/each}
			</NativeSelect>
			{#if draft.people > FREE_PEOPLE}
				<p class="text-xs text-muted-foreground">
					Empiezas con 1 agente; desde «Tu plan» pides los agentes que necesites.
				</p>
			{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="name">Tu nombre</Label>
			<Input
				id="name"
				autocomplete="name"
				bind:value={draft.name}
				aria-invalid={!!fieldError('name')}
				aria-describedby={fieldError('name') ? 'name-error' : undefined}
			/>
			{#if fieldError('name')}
				<p id="name-error" class="text-sm text-destructive">{fieldError('name')}</p>
			{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="email">Email de trabajo</Label>
			<Input
				id="email"
				type="email"
				autocomplete="email"
				bind:value={draft.email}
				aria-invalid={!!fieldError('email')}
				aria-describedby={fieldError('email') ? 'email-error' : undefined}
			/>
			{#if fieldError('email')}
				<p id="email-error" class="text-sm text-destructive">{fieldError('email')}</p>
			{/if}
		</div>
		<div class="grid gap-1.5 sm:col-span-2">
			<Label for="password">Contraseña</Label>
			<Input
				id="password"
				type="password"
				autocomplete="new-password"
				bind:value={draft.password}
				aria-invalid={!!fieldError('password')}
				aria-describedby={fieldError('password') ? 'password-error' : 'password-hint'}
			/>
			{#if fieldError('password')}
				<p id="password-error" class="text-sm text-destructive">{fieldError('password')}</p>
			{:else}
				<p id="password-hint" class="text-xs text-muted-foreground">Al menos 8 caracteres.</p>
			{/if}
		</div>
		<div class="sm:col-span-2">
			<label class="flex items-start gap-2 text-sm">
				<input
					type="checkbox"
					class="mt-0.5 size-4 accent-amber-500"
					bind:checked={draft.accept_terms}
					aria-invalid={!!fieldError('accept_terms')}
				/>
				<span>
					Acepto los
					<a
						href={resolve('/terminos')}
						target="_blank"
						class="text-primary underline dark:text-honey">términos de servicio</a
					>
					y el
					<a
						href={resolve('/privacidad')}
						target="_blank"
						class="text-primary underline dark:text-honey">aviso de privacidad</a
					>.
				</span>
			</label>
			{#if fieldError('accept_terms')}
				<p class="mt-1 text-sm text-destructive">{fieldError('accept_terms')}</p>
			{/if}
		</div>
		<Button type="submit" disabled={sending} class="w-full sm:col-span-2" size="lg">
			{sending ? 'Creando tu cuenta…' : 'Crear cuenta gratis'}
		</Button>
	</form>

	{#snippet footer()}
		¿Ya tienes cuenta?&nbsp;<a
			href={resolve('/login')}
			class="text-primary underline dark:text-honey">Inicia sesión</a
		>
	{/snippet}
</AuthCard>
