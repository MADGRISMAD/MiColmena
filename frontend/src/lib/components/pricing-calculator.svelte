<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import { Button } from '#lib/components/ui/button/index.js';
	import {
		ANNUAL_MONTHS_PAID,
		estimate,
		FREE_PEOPLE,
		MAX_AGENTS,
		money,
		PEOPLE_STOPS,
		peopleLabel
	} from '#lib/pricing.js';
	import { cn } from '#lib/utils.js';

	let {
		agents = $bindable(5),
		people = $bindable(100),
		annual = $bindable(true),
		onrequest
	}: {
		agents?: number;
		people?: number;
		annual?: boolean;
		/** Pedir demo o cotización con estos números. */
		onrequest: () => void;
	} = $props();

	// El slider de personas se mueve por escalones (5, 10, 25… 1,000, más de 1,000).
	let peopleIndex = $state(Math.max(0, PEOPLE_STOPS.indexOf(people)));
	const lastPeople = PEOPLE_STOPS.length - 1;

	function setAgents(value: number) {
		agents = value;
		// Los agentes son parte de la empresa: si no caben, sube el tamaño.
		while (PEOPLE_STOPS[peopleIndex] < agents && peopleIndex < lastPeople) peopleIndex++;
		people = PEOPLE_STOPS[peopleIndex];
	}

	function setPeople(index: number) {
		peopleIndex = index;
		people = PEOPLE_STOPS[index];
		if (agents > people) agents = people;
	}

	const e = $derived(estimate(agents, people));
	const total = $derived(annual ? e.monthlyAnnual : e.monthly);
	const agentsText = $derived(agents > MAX_AGENTS ? `Más de ${MAX_AGENTS}` : String(agents));
	const fill = (value: number, max: number, min = 0) => `${((value - min) / (max - min)) * 100}%`;
</script>

<div
	class="grid overflow-hidden rounded-3xl border bg-card shadow-xl shadow-amber-900/5 lg:grid-cols-[1.25fr_1fr]"
>
	<!-- Controles -->
	<div class="grid content-start gap-9 p-7 sm:p-10">
		<div>
			<div class="flex items-end justify-between gap-4">
				<label for="calc-agents" class="font-semibold">
					Agentes
					<span class="block text-sm font-normal text-muted-foreground">
						Personas de tu equipo que atienden tickets
					</span>
				</label>
				<output for="calc-agents" class="text-2xl font-bold tabular-nums sm:text-3xl"
					>{agentsText}</output
				>
			</div>
			<input
				id="calc-agents"
				type="range"
				min="1"
				max={MAX_AGENTS + 1}
				step="1"
				value={agents}
				oninput={(ev) => setAgents(Number(ev.currentTarget.value))}
				aria-valuetext={`${agentsText} agentes`}
				class="range mt-4"
				style:--fill={fill(agents, MAX_AGENTS + 1, 1)}
			/>
			<div class="mt-1 flex justify-between text-xs text-muted-foreground">
				<span>1</span><span>{MAX_AGENTS}+</span>
			</div>
		</div>

		<div>
			<div class="flex items-end justify-between gap-4">
				<label for="calc-people" class="font-semibold">
					Personas en tu empresa
					<span class="block text-sm font-normal text-muted-foreground">
						Todo el personal, no solo soporte
					</span>
				</label>
				<output for="calc-people" class="text-right text-2xl font-bold tabular-nums sm:text-3xl">
					{peopleLabel(people)}
				</output>
			</div>
			<input
				id="calc-people"
				type="range"
				min="0"
				max={lastPeople}
				step="1"
				value={peopleIndex}
				oninput={(ev) => setPeople(Number(ev.currentTarget.value))}
				aria-valuetext={`${peopleLabel(people)} personas`}
				class="range mt-4"
				style:--fill={fill(peopleIndex, lastPeople)}
			/>
			<div class="mt-1 flex justify-between text-xs text-muted-foreground">
				<span>{PEOPLE_STOPS[0]}</span><span>1,000+</span>
			</div>
		</div>

		<div
			class="inline-flex w-fit rounded-full border bg-muted p-1 text-sm"
			role="group"
			aria-label="Forma de pago"
		>
			{#each [{ value: false, label: 'Mensual' }, { value: true, label: 'Anual' }] as option (option.label)}
				<button
					type="button"
					aria-pressed={annual === option.value}
					onclick={() => (annual = option.value)}
					class={cn(
						'rounded-full px-4 py-1.5 font-medium text-muted-foreground transition-colors',
						annual === option.value && 'bg-card text-foreground shadow-sm'
					)}
				>
					{option.label}
					{#if option.value}
						<span
							class="ml-1 rounded-full bg-honey/30 px-1.5 text-xs text-amber-900 dark:text-amber-200"
						>
							{12 - ANNUAL_MONTHS_PAID} meses gratis
						</span>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	<!-- Resultado -->
	<div
		class="bg-honeycomb-dark relative flex flex-col bg-sidebar p-7 text-white sm:p-10"
		aria-live="polite"
	>
		{#if e.custom}
			<p class="text-sm text-sidebar-foreground">Tu precio</p>
			<p class="mt-2 text-4xl font-bold tracking-tight">A la medida</p>
			<p class="mt-3 text-sidebar-foreground">
				Para más de {MAX_AGENTS} agentes o más de 1,000 personas preparamos una cotización con precio
				por volumen.
			</p>
			<Button
				size="lg"
				class="mt-auto w-full bg-honey text-honey-foreground hover:bg-honey/90"
				onclick={onrequest}
			>
				Pedir cotización
			</Button>
		{:else}
			<p class="text-sm text-sidebar-foreground">Tu precio</p>
			{#if e.free}
				<p class="mt-2 flex items-end gap-2">
					<span class="text-5xl font-bold tracking-tight" data-testid="total">Gratis</span>
					<span class="pb-1.5 text-sm text-sidebar-foreground">para siempre</span>
				</p>
				<p class="mt-1 text-sm text-sidebar-foreground">
					1 agente y hasta {FREE_PEOPLE} personas, con todas las funciones.
				</p>
			{:else}
				<p class="mt-2 flex items-end gap-2">
					<span class="text-5xl font-bold tracking-tight tabular-nums" data-testid="total"
						>{money(total)}</span
					>
					<span class="pb-1.5 text-sm text-sidebar-foreground">MXN al mes</span>
				</p>
				<p class="mt-1 text-sm text-sidebar-foreground">
					{annual
						? `${money(e.annualTotal)} al año · ahorras ${money(e.monthly * 12 - e.annualTotal)}`
						: 'Más IVA.'}
				</p>
			{/if}

			<dl class="mt-7 grid gap-2.5 border-t border-white/10 pt-5 text-sm">
				<div class="flex justify-between gap-4">
					<dt class="text-sidebar-foreground">
						Empresa ({peopleLabel(people).toLowerCase()} personas)
					</dt>
					<dd class="tabular-nums">{e.companyFee ? money(e.companyFee) : 'Incluido'}</dd>
				</div>
				{#each e.agentLines as line (line.price)}
					<div class="flex justify-between gap-4">
						<dt class="text-sidebar-foreground">
							{#if line.price === 0}
								1 agente incluido
							{:else}
								{line.count}
								{line.count === 1 ? 'agente' : 'agentes'} × {money(line.price)}
							{/if}
						</dt>
						<dd class="tabular-nums">
							{line.price === 0 ? 'Gratis' : money(line.count * line.price)}
						</dd>
					</div>
				{/each}
				{#if !e.free}
					<div class="flex justify-between gap-4 border-t border-white/10 pt-2.5 font-semibold">
						<dt>{annual ? 'Al mes sin el descuento anual' : 'Total al mes'}</dt>
						<dd class="tabular-nums">{money(e.monthly)}</dd>
					</div>
				{/if}
			</dl>
			{#if !e.free}
				<p class="mt-2 text-xs text-sidebar-foreground/80">Precios en pesos mexicanos, más IVA.</p>
			{/if}

			<Button
				size="lg"
				class="mt-8 w-full bg-honey text-honey-foreground hover:bg-honey/90"
				onclick={onrequest}
			>
				{e.free ? 'Quiero el plan gratis' : 'Solicitar demo con este precio'}
			</Button>
		{/if}
		<ul class="mt-6 grid gap-1.5 text-sm text-sidebar-foreground">
			{#each ['Todas las funciones incluidas', 'Sin cobro por ticket ni por cliente'] as point (point)}
				<li class="flex items-center gap-2">
					<CheckIcon class="size-4 text-honey" aria-hidden="true" />{point}
				</li>
			{/each}
		</ul>
	</div>
</div>

<style>
	/* Slider con la parte recorrida en color miel. */
	.range {
		appearance: none;
		width: 100%;
		height: 0.5rem;
		border-radius: 999px;
		background: linear-gradient(
			to right,
			var(--honey) 0 var(--fill),
			var(--muted) var(--fill) 100%
		);
		cursor: pointer;
	}
	.range:focus-visible {
		outline: 3px solid color-mix(in oklch, var(--ring) 50%, transparent);
		outline-offset: 4px;
	}
	.range::-webkit-slider-thumb {
		appearance: none;
		width: 1.5rem;
		height: 1.5rem;
		border-radius: 999px;
		background: var(--card);
		border: 3px solid var(--honey);
		box-shadow: 0 2px 6px rgb(0 0 0 / 0.2);
	}
	.range::-moz-range-thumb {
		width: 1.25rem;
		height: 1.25rem;
		border-radius: 999px;
		background: var(--card);
		border: 3px solid var(--honey);
		box-shadow: 0 2px 6px rgb(0 0 0 / 0.2);
	}
</style>
