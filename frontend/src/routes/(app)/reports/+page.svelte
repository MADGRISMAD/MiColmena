<script lang="ts">
	import DownloadIcon from '@lucide/svelte/icons/download';
	import { toast } from 'svelte-sonner';
	import { api, type Report } from '#lib/api/index.js';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatHours } from '#lib/format.js';
	import { cn } from '#lib/utils.js';

	const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
	const iso = (d: Date) =>
		`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
	const daysAgo = (n: number) => iso(new Date(Date.now() - n * 86_400_000));

	const presets = [
		{ label: '7 días', days: 7 },
		{ label: '30 días', days: 30 },
		{ label: '90 días', days: 90 }
	];
	let from = $state(daysAgo(29));
	let to = $state(iso(new Date()));
	let report = $state<Report | null>(null);
	let error = $state<string | null>(null);
	let exporting = $state(false);

	let requestId = 0;
	$effect(() => {
		const range = { from, to, tz };
		const id = ++requestId;
		error = null;
		api
			.report(range)
			.then((r) => id === requestId && (report = r))
			.catch((err) => id === requestId && (error = err.message));
	});

	async function exportCsv() {
		exporting = true;
		try {
			const blob = await api.exportTickets({ from, to, tz });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `tickets_${from}_${to}.csv`;
			a.click();
			setTimeout(() => URL.revokeObjectURL(url), 10_000);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo exportar');
		} finally {
			exporting = false;
		}
	}

	const maxDaily = $derived(
		Math.max(1, ...(report?.daily.flatMap((d) => [d.created, d.resolved]) ?? [1]))
	);
	const maxCategory = $derived(Math.max(1, ...(report?.categories.map((c) => c.count) ?? [1])));
	const csat = $derived(
		report && report.satisfaction_good + report.satisfaction_bad > 0
			? Math.round(
					(report.satisfaction_good / (report.satisfaction_good + report.satisfaction_bad)) * 100
				)
			: null
	);
	const dayLabel = (date: string) =>
		new Date(date + 'T12:00:00').toLocaleDateString('es', { day: 'numeric', month: 'short' });

	let hover = $state<number | null>(null);
	let showTable = $state(false);
</script>

<PageHeader
	eyebrow="Recursos"
	title="Reportes"
	description="Cómo va el soporte en el periodo elegido."
>
	{#snippet actions()}
		<Button variant="outline" disabled={exporting} onclick={exportCsv}>
			<DownloadIcon aria-hidden="true" />
			{exporting ? 'Exportando…' : 'Exportar CSV'}
		</Button>
	{/snippet}
</PageHeader>

<!-- Filtros en una sola fila sobre los gráficos -->
<div class="mb-6 flex flex-wrap items-end gap-3">
	<div class="inline-flex gap-1 rounded-lg bg-muted p-1" role="group" aria-label="Periodo">
		{#each presets as p (p.days)}
			{@const active = from === daysAgo(p.days - 1) && to === iso(new Date())}
			<button
				type="button"
				aria-pressed={active}
				class={cn(
					'rounded-md px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground',
					active && 'bg-card text-foreground shadow-sm'
				)}
				onclick={() => {
					from = daysAgo(p.days - 1);
					to = iso(new Date());
				}}
			>
				{p.label}
			</button>
		{/each}
	</div>
	<label class="grid gap-1 text-xs text-muted-foreground">
		Desde
		<Input type="date" bind:value={from} max={to} class="h-9" />
	</label>
	<label class="grid gap-1 text-xs text-muted-foreground">
		Hasta
		<Input type="date" bind:value={to} min={from} class="h-9" />
	</label>
</div>

{#if error}
	<Card.Root><Card.Content class="text-muted-foreground">{error}</Card.Content></Card.Root>
{:else if !report}
	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
		{#each { length: 4 }, i (i)}<Skeleton class="h-24" />{/each}
	</div>
{:else}
	<div class="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
		{#each [{ label: 'Tickets creados', value: String(report.created), hint: `${report.resolved} resueltos` }, { label: 'Primera respuesta media', value: formatHours(report.avg_first_response_hours), hint: report.first_response_sla_met === null ? 'Sin respuestas en el periodo' : `${Math.round(report.first_response_sla_met * 100)} % dentro del SLA` }, { label: 'Resolución media', value: formatHours(report.avg_resolution_hours), hint: 'Desde que se abre hasta resolverse' }, { label: 'Satisfacción', value: csat === null ? '—' : `${csat} %`, hint: `${report.satisfaction_good} buenas · ${report.satisfaction_bad} malas` }] as kpi (kpi.label)}
			<Card.Root class="gap-1 py-5">
				<Card.Content class="px-5">
					<p class="text-sm text-muted-foreground">{kpi.label}</p>
					<p class="mt-1 text-3xl font-bold tabular-nums">{kpi.value}</p>
					<p class="mt-1 text-xs text-muted-foreground">{kpi.hint}</p>
				</Card.Content>
			</Card.Root>
		{/each}
	</div>

	<Card.Root class="mb-6">
		<Card.Header class="flex flex-row flex-wrap items-center justify-between gap-2">
			<Card.Title>Volumen diario</Card.Title>
			<div class="flex items-center gap-4 text-xs text-muted-foreground">
				<span class="flex items-center gap-1.5"
					><span class="size-2.5 rounded-sm bg-[#d97706]"></span>Creados</span
				>
				<span class="flex items-center gap-1.5"
					><span class="size-2.5 rounded-sm bg-[#0284c7]"></span>Resueltos</span
				>
				<button type="button" class="underline" onclick={() => (showTable = !showTable)}>
					{showTable ? 'Ver gráfico' : 'Ver como tabla'}
				</button>
			</div>
		</Card.Header>
		<Card.Content>
			{#if showTable}
				<div class="max-h-80 overflow-y-auto">
					<table class="w-full text-sm">
						<thead class="text-left text-xs text-muted-foreground">
							<tr
								><th class="py-1 font-medium">Día</th><th class="font-medium">Creados</th><th
									class="font-medium">Resueltos</th
								></tr
							>
						</thead>
						<tbody class="tabular-nums">
							{#each report.daily as d (d.date)}
								<tr class="border-t"
									><td class="py-1">{dayLabel(d.date)}</td><td>{d.created}</td><td>{d.resolved}</td
									></tr
								>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<div class="relative">
					<!-- Línea guía con el máximo; un solo eje -->
					<div class="mb-1 text-right text-[0.7rem] text-muted-foreground tabular-nums">
						{maxDaily}
					</div>
					<div
						class="flex h-48 items-end gap-[2px] border-b border-border"
						role="img"
						aria-label={`Tickets creados y resueltos por día, del ${dayLabel(report.from)} al ${dayLabel(report.to)}`}
					>
						{#each report.daily as d, i (d.date)}
							<div
								class="relative flex h-full min-w-0 flex-1 items-end justify-center gap-px"
								role="presentation"
								onmouseenter={() => (hover = i)}
								onmouseleave={() => (hover = null)}
							>
								<span
									class="w-full max-w-2 rounded-t-[4px] bg-[#d97706]"
									style:height={`${(d.created / maxDaily) * 100}%`}
								></span>
								<span
									class="w-full max-w-2 rounded-t-[4px] bg-[#0284c7]"
									style:height={`${(d.resolved / maxDaily) * 100}%`}
								></span>
								{#if hover === i}
									<div
										class="pointer-events-none absolute bottom-full z-10 mb-2 rounded-md border bg-popover px-2.5 py-1.5 text-xs whitespace-nowrap shadow-md"
									>
										<p class="font-medium">{dayLabel(d.date)}</p>
										<p>Creados: <strong class="tabular-nums">{d.created}</strong></p>
										<p>Resueltos: <strong class="tabular-nums">{d.resolved}</strong></p>
									</div>
								{/if}
							</div>
						{/each}
					</div>
					<div class="mt-1 flex justify-between text-[0.7rem] text-muted-foreground">
						<span>{dayLabel(report.from)}</span><span>{dayLabel(report.to)}</span>
					</div>
				</div>
			{/if}
		</Card.Content>
	</Card.Root>

	<div class="grid gap-6 xl:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)]">
		<Card.Root class="gap-0 overflow-hidden pb-0">
			<Card.Header class="pb-4"><Card.Title>Por agente</Card.Title></Card.Header>
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead
						class="border-y bg-muted/50 text-left text-xs whitespace-nowrap text-muted-foreground"
					>
						<tr>
							<th class="px-4 py-2 font-medium">Agente</th>
							<th class="px-4 py-2 text-right font-medium">Pendientes</th>
							<th class="px-4 py-2 text-right font-medium">Resueltos</th>
							<th class="px-4 py-2 text-right font-medium">Resolución media</th>
							<th class="px-4 py-2 text-right font-medium">Valoración</th>
						</tr>
					</thead>
					<tbody class="tabular-nums">
						{#each report.agents as a (a.id)}
							{@const rated = a.satisfaction_good + a.satisfaction_bad}
							<tr class="border-b last:border-0">
								<td class="px-4 py-2.5">
									<span class="flex items-center gap-2">
										<HexAvatar name={a.name} id={a.id} size="xs" />{a.name}
									</span>
								</td>
								<td class="px-4 py-2.5 text-right">{a.open_assigned}</td>
								<td class="px-4 py-2.5 text-right">{a.resolved}</td>
								<td class="px-4 py-2.5 text-right">{formatHours(a.avg_resolution_hours)}</td>
								<td class="px-4 py-2.5 text-right">
									{rated ? `${Math.round((a.satisfaction_good / rated) * 100)} %` : '—'}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</Card.Root>

		<Card.Root>
			<Card.Header><Card.Title>Por categoría</Card.Title></Card.Header>
			<Card.Content>
				{#if report.categories.length === 0}
					<p class="text-sm text-muted-foreground">No hay tickets en el periodo.</p>
				{:else}
					<ul class="grid gap-3">
						{#each report.categories as c (c.name)}
							<li class="grid gap-1 text-sm" title={`${c.name}: ${c.count}`}>
								<div class="flex justify-between gap-2">
									<span class="truncate">{c.name}</span>
									<span class="text-muted-foreground tabular-nums">{c.count}</span>
								</div>
								<div class="h-2 overflow-hidden rounded-full bg-muted">
									<div
										class="h-full rounded-full bg-[#d97706]"
										style:width={`${(c.count / maxCategory) * 100}%`}
									></div>
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			</Card.Content>
		</Card.Root>
	</div>
{/if}
