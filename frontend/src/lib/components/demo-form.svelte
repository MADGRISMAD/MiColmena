<script lang="ts">
	import CircleCheckIcon from '@lucide/svelte/icons/circle-check';
	import { api, ApiError, type LeadInput } from '#lib/api/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { estimate, MAX_AGENTS, money, peopleLabel } from '#lib/pricing.js';

	let {
		agents,
		people,
		annual,
		onadjust
	}: {
		/** Lo elegido en la calculadora de precios. */
		agents: number;
		people: number;
		annual: boolean;
		/** Volver a la calculadora para cambiarlo. */
		onadjust: () => void;
	} = $props();

	let draft = $state<Omit<LeadInput, 'agents' | 'people'>>({
		name: '',
		company: '',
		email: '',
		phone: '',
		message: '',
		website: ''
	});

	const quote = $derived(estimate(agents, people));
	let errors = $state<Record<string, string>>({});
	let sending = $state(false);
	let sent = $state(false);

	function validate() {
		const e: Record<string, string> = {};
		if (!draft.name.trim()) e.name = 'Escribe tu nombre';
		if (!draft.company.trim()) e.company = 'Escribe el nombre de tu empresa';
		if (!/^\S+@\S+\.\S+$/.test(draft.email.trim())) e.email = 'Escribe un email válido';
		return e;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		errors = validate();
		if (Object.keys(errors).length) return;
		sending = true;
		try {
			await api.createLead({ ...draft, agents, people });
			sent = true;
		} catch (err) {
			errors =
				err instanceof ApiError && Object.keys(err.fields).length
					? err.fields
					: { form: err instanceof Error ? err.message : 'No se pudo enviar' };
		} finally {
			sending = false;
		}
	}
</script>

{#if sent}
	<div class="flex flex-col items-center gap-3 py-10 text-center" role="status">
		<span class="hex grid size-14 place-items-center bg-honey text-honey-foreground">
			<CircleCheckIcon class="size-7" aria-hidden="true" />
		</span>
		<p class="text-xl font-semibold">¡Gracias, {draft.name.trim().split(' ')[0]}!</p>
		<p class="max-w-sm text-muted-foreground">
			Recibimos tu solicitud. Te escribiremos a <strong>{draft.email}</strong> para agendar la demo.
		</p>
	</div>
{:else}
	<form class="grid gap-4 sm:grid-cols-2" onsubmit={submit} novalidate>
		<div
			class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-honey/50 bg-honey/10 px-4 py-3 text-sm sm:col-span-2"
		>
			<p data-testid="quote">
				<strong>{agents > MAX_AGENTS ? `Más de ${MAX_AGENTS}` : agents}</strong>
				{agents === 1 ? 'agente' : 'agentes'} · <strong>{peopleLabel(people).toLowerCase()}</strong>
				personas ·
				<strong>
					{quote.custom
						? 'cotización a la medida'
						: quote.free
							? 'plan gratis'
							: `${money(annual ? quote.monthlyAnnual : quote.monthly)} al mes`}
				</strong>
			</p>
			<button
				type="button"
				class="font-medium text-primary underline dark:text-honey"
				onclick={onadjust}
			>
				Cambiar
			</button>
		</div>
		<div class="grid gap-1.5">
			<Label for="lead-name">Nombre</Label>
			<Input
				id="lead-name"
				autocomplete="name"
				bind:value={draft.name}
				aria-invalid={!!errors.name}
				aria-describedby={errors.name ? 'lead-name-error' : undefined}
			/>
			{#if errors.name}<p id="lead-name-error" class="text-sm text-destructive">
					{errors.name}
				</p>{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="lead-company">Empresa</Label>
			<Input
				id="lead-company"
				autocomplete="organization"
				bind:value={draft.company}
				aria-invalid={!!errors.company}
				aria-describedby={errors.company ? 'lead-company-error' : undefined}
			/>
			{#if errors.company}
				<p id="lead-company-error" class="text-sm text-destructive">{errors.company}</p>
			{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="lead-email">Email de trabajo</Label>
			<Input
				id="lead-email"
				type="email"
				autocomplete="email"
				bind:value={draft.email}
				aria-invalid={!!errors.email}
				aria-describedby={errors.email ? 'lead-email-error' : undefined}
			/>
			{#if errors.email}<p id="lead-email-error" class="text-sm text-destructive">
					{errors.email}
				</p>{/if}
		</div>
		<div class="grid gap-1.5">
			<Label for="lead-phone">
				Teléfono o WhatsApp <span class="font-normal text-muted-foreground">(opcional)</span>
			</Label>
			<Input id="lead-phone" type="tel" autocomplete="tel" bind:value={draft.phone} />
		</div>
		<div class="grid gap-1.5 sm:col-span-2">
			<Label for="lead-message">
				¿Cómo atienden hoy a sus clientes? <span class="font-normal text-muted-foreground"
					>(opcional)</span
				>
			</Label>
			<Textarea
				id="lead-message"
				rows={3}
				maxlength={2000}
				bind:value={draft.message}
				placeholder="Por ejemplo: por correo y WhatsApp, y se nos pierden las solicitudes."
			/>
		</div>
		<!-- Campo trampa: oculto para personas, los bots lo llenan. -->
		<div class="hidden" aria-hidden="true">
			<label for="lead-website">Sitio web</label>
			<input id="lead-website" tabindex="-1" autocomplete="off" bind:value={draft.website} />
		</div>
		{#if errors.form}
			<p class="text-sm text-destructive sm:col-span-2" role="alert">{errors.form}</p>
		{/if}
		<div class="flex flex-wrap items-center gap-3 sm:col-span-2">
			<Button type="submit" size="lg" disabled={sending}>
				{sending ? 'Enviando…' : 'Solicitar mi demo'}
			</Button>
			<p class="text-xs text-muted-foreground">
				Sin compromiso. Solo usamos tus datos para contactarte.
			</p>
		</div>
	</form>
{/if}
