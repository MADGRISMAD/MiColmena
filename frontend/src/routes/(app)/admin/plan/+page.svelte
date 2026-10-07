<script lang="ts">
	import { toast } from 'svelte-sonner';
	import { api, ApiError } from '#lib/api/index.js';
	import PageHeader from '#lib/components/page-header.svelte';
	import PortalLink from '#lib/components/portal-link.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { estimate, MAX_AGENTS, money, PEOPLE_STOPS, peopleLabel } from '#lib/pricing.js';
	import { loadOrg, org } from '#lib/stores/org.js';
	import { cn } from '#lib/utils.js';

	$effect(() => {
		loadOrg();
	});

	const allowed = $derived($org?.max_agents ?? null);
	const current = $derived($org ? estimate(allowed ?? $org.agents, $org.people) : null);

	// Solicitud de ampliación, con el precio estimado al momento.
	let agents = $state(2);
	let people = $state(PEOPLE_STOPS[1]);
	let message = $state('');
	let sending = $state(false);
	let sent = $state(false);
	let initialized = false;
	$effect(() => {
		if (!$org || initialized) return;
		initialized = true;
		agents = Math.max(2, (allowed ?? $org.agents) + 1);
		people = PEOPLE_STOPS.find((s) => s >= $org.people) ?? PEOPLE_STOPS[PEOPLE_STOPS.length - 1];
	});
	const wanted = $derived(estimate(agents, people));

	async function request(event: SubmitEvent) {
		event.preventDefault();
		sending = true;
		try {
			await api.requestUpgrade({ agents, people, message: message.trim() });
			sent = true;
		} catch (err) {
			const fields = err instanceof ApiError ? Object.values(err.fields) : [];
			toast.error(fields[0] ?? (err instanceof Error ? err.message : 'No se pudo enviar'));
		} finally {
			sending = false;
		}
	}
</script>

<PageHeader
	eyebrow="Administración"
	title="Tu plan"
	description="Lo que incluye tu plan, lo que usas y cómo ampliarlo."
/>

{#if !$org}
	<div class="grid gap-4 lg:grid-cols-2"><Skeleton class="h-48" /><Skeleton class="h-48" /></div>
{:else}
	<div class="grid gap-6 lg:grid-cols-2">
		<Card.Root>
			<Card.Header>
				<Card.Title>
					{current?.free ? 'Plan gratis' : allowed === null ? 'Plan a la medida' : 'Plan de pago'}
				</Card.Title>
				<Card.Description>
					{current?.free
						? 'Gratis para siempre: 1 agente, hasta 10 personas y todas las funciones.'
						: current && !current.custom
							? `${money(current.monthly)} MXN al mes, más IVA (pago mensual).`
							: 'Condiciones acordadas con tu empresa.'}
				</Card.Description>
			</Card.Header>
			<Card.Content class="grid gap-5">
				<div>
					<div class="flex items-end justify-between text-sm">
						<span class="font-medium">Agentes</span>
						<span class="tabular-nums" data-testid="agents-usage">
							{$org.agents} de {allowed ?? 'ilimitados'}
						</span>
					</div>
					{#if allowed !== null}
						<div class="mt-2 h-2 overflow-hidden rounded-full bg-muted">
							<div
								class={cn(
									'h-full rounded-full',
									$org.agents >= allowed ? 'bg-amber-600' : 'bg-honey'
								)}
								style:width={`${Math.min(100, ($org.agents / allowed) * 100)}%`}
							></div>
						</div>
						{#if $org.agents >= allowed}
							<p class="mt-2 text-xs text-muted-foreground">
								Ya usas todos tus agentes. Para dar de alta a otro, pide ampliar tu plan.
							</p>
						{/if}
					{/if}
				</div>
				<div class="flex justify-between text-sm">
					<span class="font-medium">Personas en la empresa</span>
					<span>{$org.people.toLocaleString('es-MX')}</span>
				</div>
				<p class="text-xs text-muted-foreground">
					Los clientes que abren tickets no cuentan. Cada agente tiene una sola sesión a la vez.
				</p>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title>Tu portal de soporte</Card.Title>
				<Card.Description>
					Compártelo con tus clientes: ahí abren tickets y leen tus artículos de ayuda.
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<PortalLink slug={$org.slug} />
			</Card.Content>
		</Card.Root>

		<Card.Root class="lg:col-span-2">
			<Card.Header>
				<Card.Title>Ampliar tu plan</Card.Title>
				<Card.Description>
					Indica cuántos agentes necesitas y te contactamos para activarlo.
				</Card.Description>
			</Card.Header>
			<Card.Content>
				{#if sent}
					<p class="rounded-lg border border-honey/50 bg-honey/10 p-4 text-sm" role="status">
						Recibimos tu solicitud. Te escribiremos para confirmar el precio y activar los agentes.
					</p>
				{:else}
					<form class="grid gap-4 md:grid-cols-[1fr_1fr_1.3fr] md:items-start" onsubmit={request}>
						<div class="grid gap-1.5">
							<Label for="upgrade-agents">Agentes</Label>
							<Input id="upgrade-agents" type="number" min="1" max="100000" bind:value={agents} />
						</div>
						<div class="grid gap-1.5">
							<Label for="upgrade-people">Personas en la empresa</Label>
							<NativeSelect id="upgrade-people" class="w-full" bind:value={people}>
								{#each PEOPLE_STOPS as stop (stop)}
									<NativeSelectOption value={stop}>{peopleLabel(stop)}</NativeSelectOption>
								{/each}
							</NativeSelect>
						</div>
						<div class="rounded-lg bg-muted/60 p-3 text-sm" aria-live="polite">
							<p class="text-muted-foreground">Precio estimado</p>
							<p class="text-xl font-semibold" data-testid="upgrade-estimate">
								{wanted.custom || agents > MAX_AGENTS
									? 'A la medida'
									: wanted.free
										? 'Gratis'
										: `${money(wanted.monthly)} MXN al mes`}
							</p>
							<p class="text-xs text-muted-foreground">Más IVA. 2 meses gratis con pago anual.</p>
						</div>
						<div class="grid gap-1.5 md:col-span-3">
							<Label for="upgrade-message">
								Comentarios <span class="font-normal text-muted-foreground">(opcional)</span>
							</Label>
							<Textarea id="upgrade-message" rows={2} bind:value={message} maxlength={2000} />
						</div>
						<div class="md:col-span-3">
							<Button type="submit" disabled={sending || agents < 1}>
								{sending ? 'Enviando…' : 'Pedir ampliación'}
							</Button>
						</div>
					</form>
				{/if}
			</Card.Content>
		</Card.Root>
	</div>
{/if}
