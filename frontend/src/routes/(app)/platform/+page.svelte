<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from 'svelte-sonner';
	import { api, ApiError, type PlatformOrg } from '#lib/api/index.js';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatDateTime, formatRelative } from '#lib/format.js';
	import { estimate, money } from '#lib/pricing.js';
	import { user } from '#lib/stores/auth.js';
	import { cn } from '#lib/utils.js';

	let orgs = $state<PlatformOrg[] | null>(null);
	let search = $state('');

	async function load() {
		try {
			orgs = (await api.listPlatformOrgs()).items;
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) goto(resolve('/(app)/tickets'));
			orgs = [];
		}
	}
	$effect(() => {
		if ($user) load();
	});

	async function change(
		o: PlatformOrg,
		input: Parameters<typeof api.updatePlatformOrg>[1],
		msg: string
	) {
		try {
			const updated = await api.updatePlatformOrg(o.id, input);
			orgs = orgs?.map((x) => (x.id === o.id ? { ...x, ...updated } : x)) ?? null;
			toast.success(msg);
		} catch (err) {
			const fields = err instanceof ApiError ? Object.values(err.fields) : [];
			toast.error(fields[0] ?? (err instanceof Error ? err.message : 'No se pudo guardar'));
		}
	}

	function setAgents(o: PlatformOrg, value: string) {
		const n = Number(value);
		if (value.trim() === '' || n === 0)
			change(o, { unlimited: true }, `${o.name}: agentes ilimitados`);
		else if (Number.isInteger(n) && n > 0) change(o, { max_agents: n }, `${o.name}: ${n} agentes`);
	}

	const filtered = $derived(
		(orgs ?? []).filter((o) =>
			(o.name + ' ' + o.slug).toLowerCase().includes(search.trim().toLowerCase())
		)
	);
	const revenue = $derived(
		(orgs ?? [])
			.filter((o) => !o.suspended && o.max_agents !== null)
			.reduce((sum, o) => {
				const e = estimate(o.max_agents ?? 1, o.people);
				return sum + (e.custom ? 0 : e.monthly);
			}, 0)
	);
</script>

<PageHeader
	eyebrow="Plataforma"
	title="Empresas"
	description="Todas las empresas que usan BeHIve. Ajusta sus planes o suspende cuentas."
/>

{#if orgs === null}
	<Skeleton class="h-64" />
{:else}
	<div class="mb-4 grid gap-4 sm:grid-cols-3">
		{#each [{ label: 'Empresas', value: String(orgs.length) }, { label: 'En plan gratis', value: String(orgs.filter((o) => o.max_agents === 1 && o.people <= 10).length) }, { label: 'Ingreso mensual estimado', value: money(revenue) }] as kpi (kpi.label)}
			<Card.Root class="py-4">
				<Card.Content class="px-5">
					<p class="text-sm text-muted-foreground">{kpi.label}</p>
					<p class="mt-1 text-2xl font-semibold tabular-nums">{kpi.value}</p>
				</Card.Content>
			</Card.Root>
		{/each}
	</div>

	<Input
		bind:value={search}
		placeholder="Buscar empresa"
		aria-label="Buscar empresa"
		class="mb-4 max-w-xs"
	/>

	<Card.Root class="gap-0 overflow-x-auto py-0">
		<table class="w-full text-sm">
			<thead class="border-b bg-muted/50 text-left text-xs whitespace-nowrap text-muted-foreground">
				<tr>
					<th class="px-4 py-2.5 font-medium">Empresa</th>
					<th class="px-4 py-2.5 font-medium">Agentes</th>
					<th class="px-4 py-2.5 font-medium">Personas</th>
					<th class="px-4 py-2.5 font-medium">Clientes</th>
					<th class="px-4 py-2.5 font-medium">Tickets</th>
					<th class="px-4 py-2.5 font-medium">Actividad</th>
					<th class="px-4 py-2.5 font-medium">Precio</th>
					<th class="px-4 py-2.5"><span class="sr-only">Acciones</span></th>
				</tr>
			</thead>
			<tbody>
				{#each filtered as o (o.id)}
					{@const e = estimate(o.max_agents ?? o.agents, o.people)}
					<tr class={cn('border-b last:border-0', o.suspended && 'bg-red-50/60 dark:bg-red-500/5')}>
						<td class="px-4 py-3">
							<p class="font-medium">{o.name}</p>
							<p class="font-mono text-xs text-muted-foreground">
								/e/{o.slug} · desde {formatDateTime(o.created_at)}
							</p>
						</td>
						<td class="px-4 py-3">
							<label class="flex items-center gap-1.5 whitespace-nowrap">
								<span class="tabular-nums">{o.agents} /</span>
								<input
									type="number"
									min="0"
									class="h-8 w-16 rounded-md border bg-transparent px-2 tabular-nums"
									value={o.max_agents ?? ''}
									placeholder="∞"
									aria-label={`Agentes permitidos para ${o.name} (vacío = sin límite)`}
									disabled={o.id === 1}
									onchange={(ev) => setAgents(o, ev.currentTarget.value)}
								/>
							</label>
						</td>
						<td class="px-4 py-3">
							<input
								type="number"
								min="1"
								class="h-8 w-20 rounded-md border bg-transparent px-2 tabular-nums"
								value={o.people}
								aria-label={`Personas en ${o.name}`}
								onchange={(ev) => {
									const n = Number(ev.currentTarget.value);
									if (n > 0) change(o, { people: n }, `${o.name}: ${n} personas`);
								}}
							/>
						</td>
						<td class="px-4 py-3 tabular-nums">{o.customers}</td>
						<td class="px-4 py-3 tabular-nums">{o.tickets}</td>
						<td class="px-4 py-3 whitespace-nowrap text-muted-foreground">
							{o.last_ticket_at ? formatRelative(o.last_ticket_at) : '—'}
						</td>
						<td class="px-4 py-3 whitespace-nowrap">
							{o.id === 1
								? 'Plataforma'
								: e.free
									? 'Gratis'
									: e.custom || o.max_agents === null
										? 'A la medida'
										: money(e.monthly)}
						</td>
						<td class="px-4 py-3 text-right">
							{#if o.id !== 1}
								<Button
									size="sm"
									variant={o.suspended ? 'outline' : 'ghost'}
									onclick={() => {
										if (
											o.suspended ||
											confirm(`¿Suspender ${o.name}? Nadie de esa empresa podrá entrar.`)
										)
											change(
												o,
												{ suspended: !o.suspended },
												o.suspended ? 'Empresa reactivada' : 'Empresa suspendida'
											);
									}}
								>
									{o.suspended ? 'Reactivar' : 'Suspender'}
								</Button>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</Card.Root>
{/if}
