<script lang="ts">
	import MailIcon from '@lucide/svelte/icons/mail';
	import PhoneIcon from '@lucide/svelte/icons/phone';
	import { toast } from 'svelte-sonner';
	import { api, type Lead } from '#lib/api/index.js';
	import EmptyState from '#lib/components/empty-state.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatDateTime, formatRelative } from '#lib/format.js';
	import { onLive } from '#lib/live/index.js';
	import { estimate, MAX_AGENTS, money, peopleLabel } from '#lib/pricing.js';
	import { cn } from '#lib/utils.js';

	let leads = $state<Lead[] | null>(null);

	async function load() {
		leads = await api
			.listLeads()
			.then((r) => r.items)
			.catch(() => []);
	}
	$effect(() => {
		load();
	});
	// Una solicitud nueva llega como notificación a los administradores.
	$effect(() => onLive((e) => e.type === 'notification' && load()));

	async function toggle(lead: Lead) {
		try {
			const updated = await api.setLeadHandled(lead.id, !lead.handled);
			leads = leads?.map((l) => (l.id === lead.id ? updated : l)) ?? null;
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo guardar');
		}
	}

	// El precio se recalcula con la fórmula actual; no se guarda lo que vio la persona.
	function quote(lead: Lead): string {
		const e = estimate(lead.agents, lead.people);
		if (e.custom) return 'Cotización a la medida';
		if (e.free) return 'Plan gratis';
		return `${money(e.monthly)} al mes (pago mensual)`;
	}
	const pending = $derived(leads?.filter((l) => !l.handled).length ?? 0);
</script>

<PageHeader
	eyebrow="Administración"
	title="Solicitudes de demo"
	description={`Empresas que pidieron una demo desde la página principal. ${pending} por atender.`}
/>

{#if leads === null}
	<div class="grid gap-3"><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
{:else if leads.length === 0}
	<Card.Root>
		<EmptyState
			title="Aún no hay solicitudes"
			description="Cuando alguien llene el formulario de la página principal, aparecerá aquí y te llegará un aviso."
		/>
	</Card.Root>
{:else}
	<ul class="grid gap-3">
		{#each leads as lead (lead.id)}
			<li>
				<Card.Root class={cn('py-4', lead.handled && 'opacity-60')}>
					<Card.Content class="flex-row flex-wrap items-start gap-4 px-5">
						<div class="min-w-0 flex-1">
							<p class="flex flex-wrap items-center gap-2 font-semibold">
								{lead.company}
								{#if lead.agents > 0}
									<span class="rounded bg-honey/20 px-1.5 text-xs font-medium">
										{lead.agents > MAX_AGENTS ? `Más de ${MAX_AGENTS}` : lead.agents} agentes ·
										{peopleLabel(lead.people).toLowerCase()} personas · {quote(lead)}
									</span>
								{:else if lead.team_size}
									<span class="rounded bg-muted px-1.5 text-xs font-normal text-muted-foreground">
										{lead.team_size} agentes
									</span>
								{/if}
							</p>
							<p class="mt-1 text-sm">{lead.name}</p>
							<p class="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground">
								<a
									href={`mailto:${lead.email}`}
									class="inline-flex items-center gap-1 hover:text-foreground"
								>
									<MailIcon class="size-3.5" aria-hidden="true" />{lead.email}
								</a>
								{#if lead.phone}
									<a
										href={`tel:${lead.phone.replace(/[^\d+]/g, '')}`}
										class="inline-flex items-center gap-1 hover:text-foreground"
									>
										<PhoneIcon class="size-3.5" aria-hidden="true" />{lead.phone}
									</a>
								{/if}
							</p>
							{#if lead.message}
								<p class="mt-2 rounded-md bg-muted p-2 text-sm whitespace-pre-wrap">
									{lead.message}
								</p>
							{/if}
						</div>
						<div class="flex flex-col items-end gap-2">
							<time
								class="text-xs text-muted-foreground"
								datetime={lead.created_at}
								title={formatDateTime(lead.created_at)}
							>
								{formatRelative(lead.created_at)}
							</time>
							<Button
								size="sm"
								variant={lead.handled ? 'ghost' : 'outline'}
								onclick={() => toggle(lead)}
							>
								{lead.handled ? 'Marcar pendiente' : 'Marcar atendida'}
							</Button>
						</div>
					</Card.Content>
				</Card.Root>
			</li>
		{/each}
	</ul>
{/if}
