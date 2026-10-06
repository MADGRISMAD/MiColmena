<script lang="ts">
	import { resolve } from '$app/paths';
	import ArrowRightIcon from '@lucide/svelte/icons/arrow-right';
	import ChartColumnIcon from '@lucide/svelte/icons/chart-column';
	import ChevronsUpIcon from '@lucide/svelte/icons/chevrons-up';
	import InboxIcon from '@lucide/svelte/icons/inbox';
	import LockIcon from '@lucide/svelte/icons/lock';
	import SearchIcon from '@lucide/svelte/icons/search';
	import ShieldCheckIcon from '@lucide/svelte/icons/shield-check';
	import type { TicketPriority, TicketStatus } from '#lib/api/types.js';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import Logo from '#lib/components/logo.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { isAuthenticated } from '#lib/stores/auth.js';

	const features = [
		{
			icon: InboxIcon,
			title: 'Bandeja compartida',
			text: 'Todas las solicitudes en un solo lugar, con vistas para lo tuyo, lo que nadie ha tomado y lo que espera al cliente.'
		},
		{
			icon: LockIcon,
			title: 'Notas internas',
			text: 'El equipo se coordina dentro del ticket. Las notas se ven en ámbar y el cliente nunca las recibe.'
		},
		{
			icon: ChevronsUpIcon,
			title: 'Prioridades a la vista',
			text: 'Estados con color e iconos de prioridad: lo urgente salta a la vista aunque la lista sea larga.'
		},
		{
			icon: SearchIcon,
			title: 'Búsqueda en español',
			text: 'Encuentra «impresora» aunque escribas «impresoras». Pulsa / desde cualquier pantalla para buscar.'
		},
		{
			icon: ChartColumnIcon,
			title: 'Dashboard útil',
			text: 'Pendientes, sin asignar, urgentes y tiempo de resolución, con tu lista de trabajo al lado.'
		},
		{
			icon: ShieldCheckIcon,
			title: 'Seguro desde el inicio',
			text: 'Roles para clientes, agentes y administradores, y bloqueo automático ante intentos de acceso repetidos.'
		}
	];

	const steps = [
		{
			title: 'El cliente abre un ticket',
			text: 'Describe su problema desde la web y sigue cada respuesta.'
		},
		{
			title: 'El equipo lo toma',
			text: 'Se asigna, se prioriza y se responde; las notas internas quedan entre agentes.'
		},
		{
			title: 'Se resuelve y queda el historial',
			text: 'Todo el contexto en el ticket, listo para la próxima vez.'
		}
	];

	// Datos de ejemplo para la vista previa del producto (solo ilustración).
	const preview: {
		id: number;
		title: string;
		status: TicketStatus;
		priority: TicketPriority;
		who: string;
	}[] = [
		{
			id: 128,
			title: 'Error 500 al exportar a PDF',
			status: 'open',
			priority: 'urgent',
			who: 'Marta Soto'
		},
		{
			id: 127,
			title: 'La impresora no imprime en color',
			status: 'in_progress',
			priority: 'high',
			who: 'Luis Ramírez'
		},
		{
			id: 126,
			title: 'Factura duplicada de septiembre',
			status: 'waiting',
			priority: 'medium',
			who: 'Marta Soto'
		},
		{
			id: 125,
			title: 'Alta de usuario para ventas',
			status: 'resolved',
			priority: 'low',
			who: 'Luis Ramírez'
		}
	];

	const headline = 'El soporte de tu equipo, organizado como una'.split(' ');

	// Hexágonos del parallax: x/y en %, s = tamaño en px, d = profundidad (más alto = más cerca y más rápido).
	const hexes = [
		{ x: 4, y: 14, s: 64, d: 1.2, o: 0.9 },
		{ x: 12, y: 78, s: 26, d: 0.35, o: 0.35 },
		{ x: 44, y: 6, s: 18, d: 0.5, o: 0.4 },
		{ x: 52, y: 90, s: 40, d: 1.4, o: 0.7 },
		{ x: 86, y: 8, s: 46, d: 0.6, o: 0.5 },
		{ x: 95, y: 58, s: 80, d: 1.6, o: 0.8 },
		{ x: 30, y: 46, s: 14, d: 0.25, o: 0.3 },
		{ x: 72, y: 70, s: 22, d: 0.45, o: 0.35 },
		{ x: 62, y: 22, s: 12, d: 0.2, o: 0.3 }
	];

	const marquee = [
		'Bandeja compartida',
		'Notas internas',
		'SLA por prioridad',
		'Menciones',
		'Base de conocimiento',
		'Reportes en CSV',
		'Tiempo real',
		'Encuestas de satisfacción'
	];

	// Las capas del hero siguen al mouse; con touch no hace falta.
	function tilt(e: PointerEvent & { currentTarget: HTMLElement }) {
		if (e.pointerType !== 'mouse') return;
		const r = e.currentTarget.getBoundingClientRect();
		e.currentTarget.style.setProperty('--mx', ((e.clientX - r.left) / r.width - 0.5).toFixed(3));
		e.currentTarget.style.setProperty('--my', ((e.clientY - r.top) / r.height - 0.5).toFixed(3));
	}

	function untilt(e: PointerEvent & { currentTarget: HTMLElement }) {
		e.currentTarget.style.setProperty('--mx', '0');
		e.currentTarget.style.setProperty('--my', '0');
	}
</script>

<svelte:head>
	<title>MiColmena · Mesa de ayuda para tu equipo de soporte</title>
	<meta
		name="description"
		content="MiColmena organiza los tickets de soporte de tu equipo: bandeja compartida, notas internas, prioridades y dashboard."
	/>
</svelte:head>

<div class="landing flex min-h-screen flex-col">
	<header class="sticky top-0 z-30 border-b bg-background/80 backdrop-blur">
		<div class="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4">
			<a href={resolve('/')} aria-label="MiColmena, inicio"><Logo /></a>
			<nav
				class="hidden items-center gap-6 text-sm text-muted-foreground md:flex"
				aria-label="Secciones"
			>
				<!-- eslint-disable svelte/no-navigation-without-resolve -- anclas dentro de esta página -->
				<a href="#funciones" class="hover:text-foreground">Funciones</a>
				<a href="#como-funciona" class="hover:text-foreground">Cómo funciona</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			</nav>
			<div class="flex items-center gap-2">
				{#if $isAuthenticated}
					<Button href={resolve('/(app)/tickets')}>Ir a mis tickets</Button>
				{:else}
					<Button variant="ghost" href={resolve('/login')}>Iniciar sesión</Button>
					<Button href={resolve('/register')} class="hidden sm:inline-flex">Crear cuenta</Button>
				{/if}
			</div>
		</div>
		<div class="progress absolute inset-x-0 -bottom-px h-0.5 bg-honey" aria-hidden="true"></div>
	</header>

	<main class="flex-1">
		<!-- Hero: capas de parallax (scroll + mouse) detrás del contenido -->
		<!-- El mouse solo mueve decoración. -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<section
			class="hero relative flex items-center overflow-hidden lg:sticky lg:top-16 lg:h-[calc(100svh-4rem)]"
			onpointermove={tilt}
			onpointerleave={untilt}
		>
			<div
				class="layer bg-honeycomb absolute inset-x-0 -inset-y-1/4"
				style="--d: -0.3"
				aria-hidden="true"
			></div>
			<div
				class="layer glow absolute -top-40 left-1/2 -ml-[20rem] size-[40rem] rounded-full bg-honey/30 blur-3xl"
				style="--d: 0.15"
				aria-hidden="true"
			></div>
			{#each hexes as h, i (i)}
				<div
					class="layer absolute"
					style="left: {h.x}%; top: {h.y}%; --d: {h.d}"
					aria-hidden="true"
				>
					<span
						class="floater hex block bg-honey"
						style="width: {h.s}px; height: {h.s *
							1.15}px; opacity: {h.o}; --i: {i}; filter: blur({h.d < 0.4 ? 2 : 0}px)"
					></span>
				</div>
			{/each}

			<div
				class="hero-inner relative mx-auto grid w-full max-w-6xl items-center gap-12 px-4 py-16 sm:py-24 lg:grid-cols-[1fr_1.1fr] lg:py-0"
			>
				<div class="layer" style="--d: 0.2">
					<p
						class="enter inline-flex items-center gap-2 rounded-full border bg-card/80 px-3 py-1 text-xs font-medium text-muted-foreground shadow-xs backdrop-blur"
					>
						<span class="hex pulse size-2.5 bg-honey" aria-hidden="true"></span>
						Mesa de ayuda para equipos de soporte
					</p>
					<h1 class="mt-5 text-4xl font-bold tracking-tight text-balance sm:text-5xl lg:text-6xl">
						{#each headline as word, i (i)}
							<span class="word mr-[0.25em]" style="--i: {i}">{word}</span>
						{/each}
						<span
							class="word relative whitespace-nowrap text-primary dark:text-honey"
							style="--i: {headline.length}"
						>
							colmena
							<span
								class="underline-draw absolute inset-x-0 -bottom-1 h-2 rounded-full bg-honey/60"
								aria-hidden="true"
							></span>
						</span>
					</h1>
					<p
						class="enter mt-6 max-w-xl text-lg text-pretty text-muted-foreground"
						style="--delay: 700ms"
					>
						Recibe las solicitudes de tus clientes, repártelas en el equipo y resuélvelas con todo
						el historial en un mismo lugar. Sencillo de usar desde el primer día.
					</p>
					<div class="enter mt-8 flex flex-wrap gap-3" style="--delay: 850ms">
						<Button size="lg" href={resolve('/register')}>
							Crear una cuenta
							<ArrowRightIcon aria-hidden="true" />
						</Button>
						<Button size="lg" variant="outline" href={resolve('/login')}>Ya tengo cuenta</Button>
					</div>
				</div>

				<!-- Vista previa del producto, hecha con los mismos componentes de la app -->
				<div class="layer relative mb-16 lg:mb-0" style="--d: 0.6" aria-hidden="true">
					<div class="enter perspective-[1400px]" style="--delay: 300ms">
						<div class="preview-tilt">
							<div class="overflow-hidden rounded-xl border bg-card shadow-2xl shadow-amber-900/20">
								<div class="flex items-center gap-1.5 border-b bg-muted/60 px-4 py-2.5">
									<span class="size-2.5 rounded-full bg-red-400/70"></span>
									<span class="size-2.5 rounded-full bg-amber-400/70"></span>
									<span class="size-2.5 rounded-full bg-emerald-400/70"></span>
									<span class="ml-3 text-xs text-muted-foreground">Todos los tickets</span>
								</div>
								<div class="flex">
									<div
										class="bg-honeycomb-dark hidden w-14 shrink-0 flex-col items-center gap-3 bg-sidebar py-4 sm:flex"
									>
										<span class="hex size-7 bg-honey"></span>
										<span class="size-5 rounded bg-white/15"></span>
										<span class="size-5 rounded bg-white/10"></span>
										<span class="size-5 rounded bg-white/10"></span>
									</div>
									<ul class="min-w-0 flex-1 divide-y text-sm">
										{#each preview as row, i (row.id)}
											<li class="row flex items-center gap-3 px-4 py-3" style="--i: {i}">
												<PriorityBadge priority={row.priority} compact />
												<span class="font-mono text-xs text-muted-foreground">#{row.id}</span>
												<span class="min-w-0 flex-1 truncate font-medium">{row.title}</span>
												<StatusBadge status={row.status} class="hidden sm:inline-flex" />
												<HexAvatar name={row.who} size="xs" />
											</li>
										{/each}
									</ul>
								</div>
							</div>
						</div>
					</div>
					<div class="layer absolute -right-3 -bottom-20 hidden sm:block" style="--d: 0.9">
						<div
							class="bob w-64 rotate-1 rounded-xl border border-dashed border-amber-400 bg-amber-50 p-3 text-sm shadow-lg dark:bg-amber-950"
						>
							<p
								class="flex items-center gap-1.5 text-xs font-medium text-amber-700 dark:text-amber-400"
							>
								<LockIcon class="size-3" /> Nota interna
							</p>
							<p class="mt-1">Pedir el cartucho cian al proveedor antes del viernes.</p>
						</div>
					</div>
				</div>
			</div>

			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- ancla dentro de esta página -->
			<a
				href="#funciones"
				class="scroll-cue absolute bottom-6 left-1/2 hidden -translate-x-1/2 lg:block"
				aria-label="Bajar a funciones"
			>
				<span class="flex h-9 w-6 justify-center rounded-full border-2 border-foreground/30 pt-1.5">
					<span class="cue-dot size-1.5 rounded-full bg-foreground/50"></span>
				</span>
			</a>
		</section>

		<!-- Telón: el resto de la página sube por encima del hero -->
		<div
			class="curtain relative z-10 rounded-t-[2rem] border-t bg-background shadow-[0_-24px_60px_-24px_rgb(120_53_15/0.25)]"
		>
			<div class="overflow-hidden border-b py-5" aria-hidden="true">
				<div class="marquee flex w-max gap-8 text-sm font-medium text-muted-foreground">
					{#each [...marquee, ...marquee] as item, i (i)}
						<span class="flex items-center gap-8 whitespace-nowrap">
							<span class="hex size-2 bg-honey"></span>{item}
						</span>
					{/each}
				</div>
			</div>

			<!-- Funciones -->
			<section id="funciones" class="mx-auto max-w-6xl scroll-mt-20 px-4 py-24">
				<div class="reveal max-w-2xl">
					<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">
						Todo lo que tu equipo necesita, nada que estorbe
					</h2>
					<p class="mt-3 text-muted-foreground">
						Las herramientas de soporte grandes hacen de todo; MiColmena se enfoca en que atender a
						tus clientes sea rápido y ordenado.
					</p>
				</div>
				<div class="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{#each features as feature, i (feature.title)}
						<div
							class="reveal group rounded-xl border bg-card p-6 shadow-xs transition-[box-shadow,translate] duration-300 hover:-translate-y-1 hover:shadow-lg hover:shadow-amber-900/10"
							style="--i: {i % 3}"
						>
							<span
								class="hex grid size-11 place-items-center bg-honey text-honey-foreground transition-transform duration-500 group-hover:scale-110 group-hover:rotate-[30deg]"
							>
								<feature.icon
									class="size-5 transition-transform duration-500 group-hover:-rotate-[30deg]"
									aria-hidden="true"
								/>
							</span>
							<h3 class="mt-4 font-semibold">{feature.title}</h3>
							<p class="mt-2 text-sm text-muted-foreground">{feature.text}</p>
						</div>
					{/each}
				</div>
			</section>

			<!-- Cómo funciona -->
			<section
				id="como-funciona"
				class="bg-honeycomb-dark relative scroll-mt-20 overflow-hidden bg-sidebar text-white"
			>
				<div
					class="layer-view pointer-events-none absolute -top-40 -right-40 size-[32rem] rounded-full bg-honey/15 blur-3xl"
					aria-hidden="true"
				></div>
				<div class="relative mx-auto max-w-6xl px-4 py-24">
					<h2 class="reveal text-3xl font-bold tracking-tight sm:text-4xl">Cómo funciona</h2>
					<ol class="relative mt-14 grid gap-10 md:grid-cols-3">
						<div
							class="steps-line absolute top-6 right-[16%] left-6 hidden h-0.5 bg-honey md:block"
							aria-hidden="true"
						></div>
						{#each steps as step, i (step.title)}
							<li class="step relative" style="--i: {i}">
								<span
									class="hex grid size-12 place-items-center bg-honey text-lg font-bold text-honey-foreground"
								>
									{i + 1}
								</span>
								<h3 class="mt-4 font-semibold">{step.title}</h3>
								<p class="mt-2 text-sm text-sidebar-foreground">{step.text}</p>
							</li>
						{/each}
					</ol>
				</div>
			</section>

			<!-- Llamada final -->
			<section class="mx-auto max-w-6xl px-4 py-24">
				<div
					class="cta bg-honeycomb-dark relative overflow-hidden rounded-2xl bg-sidebar px-8 py-16 text-center"
				>
					<div
						class="glow-pulse pointer-events-none absolute -bottom-32 left-1/2 -ml-[15rem] size-[30rem] rounded-full bg-honey/25 blur-3xl"
						aria-hidden="true"
					></div>
					<h2 class="relative text-3xl font-bold tracking-tight text-white sm:text-4xl">
						Pon orden en tu soporte
					</h2>
					<p class="relative mx-auto mt-3 max-w-lg text-sidebar-foreground">
						Crea tu cuenta y abre tu primer ticket en un par de minutos.
					</p>
					<Button
						size="lg"
						href={resolve('/register')}
						class="relative mt-8 bg-honey text-honey-foreground hover:bg-honey/90"
					>
						Crear una cuenta
						<ArrowRightIcon aria-hidden="true" />
					</Button>
				</div>
			</section>
		</div>
	</main>

	<footer class="relative z-10 border-t bg-background">
		<div
			class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 px-4 py-8 text-sm text-muted-foreground"
		>
			<Logo class="scale-90" />
			<p>© {new Date().getFullYear()} MiColmena</p>
		</div>
	</footer>
</div>

<style>
	/* Todo el movimiento respeta "reducir movimiento"; sin él la página queda estática y completa. */
	:global(html:has(.landing)) {
		scroll-behavior: smooth;
	}

	.progress {
		transform: scaleX(0);
		transform-origin: left;
	}

	@media (prefers-reduced-motion: reduce) {
		:global(html:has(.landing)) {
			scroll-behavior: auto;
		}
	}

	@media (prefers-reduced-motion: no-preference) {
		/* --- Mouse: cada capa se desplaza según su profundidad --- */
		.layer {
			translate: calc(var(--mx, 0) * var(--d) * -48px) calc(var(--my, 0) * var(--d) * -32px);
			transition: translate 0.8s cubic-bezier(0.2, 0.8, 0.2, 1);
		}

		/* --- Hero cinético al cargar --- */
		.word {
			display: inline-block;
			animation: word-in 0.9s cubic-bezier(0.2, 0.8, 0.2, 1) both;
			animation-delay: calc(var(--i) * 60ms);
		}
		.underline-draw {
			transform-origin: left;
			animation: draw 0.7s cubic-bezier(0.65, 0, 0.35, 1) 0.9s both;
		}
		.enter {
			animation: rise 1s cubic-bezier(0.2, 0.8, 0.2, 1) var(--delay, 0ms) both;
		}
		.row {
			animation: slide-in 0.6s cubic-bezier(0.2, 0.8, 0.2, 1) both;
			animation-delay: calc(700ms + var(--i) * 110ms);
		}
		.floater {
			animation: float 9s ease-in-out infinite alternate;
			animation-delay: calc(var(--i) * -1.3s);
		}
		.bob {
			animation: bob 5s ease-in-out infinite alternate;
		}
		.pulse {
			animation: pulse 2s ease-in-out infinite;
		}
		.cue-dot {
			animation: cue 1.6s ease-in-out infinite;
		}
		.marquee {
			animation: marquee 40s linear infinite;
		}
		.marquee:hover {
			animation-play-state: paused;
		}
		.glow-pulse {
			animation: glow 6s ease-in-out infinite alternate;
		}

		/* --- Ligado al scroll (Chrome, Edge, Safari 26+; en otros queda estático) --- */
		@supports (animation-timeline: scroll()) {
			.progress {
				animation: grow linear both;
				animation-timeline: scroll(root);
			}

			/* Parallax multicapa: cada capa sube a distinta velocidad. */
			.hero .layer {
				animation: drift linear both;
				animation-timeline: scroll(root);
				animation-range: 0 100vh;
			}
			.hero-inner {
				animation: hero-out linear both;
				animation-timeline: scroll(root);
				animation-range: 0 90vh;
			}
			.preview-tilt {
				animation: untilt linear both;
				animation-timeline: scroll(root);
				animation-range: 0 60vh;
			}
			.scroll-cue {
				animation: fade-out linear both;
				animation-timeline: scroll(root);
				animation-range: 0 20vh;
			}

			/* Transiciones entre secciones: aparecen al entrar en pantalla. */
			.reveal {
				animation: reveal linear both;
				animation-timeline: view();
				animation-range: entry calc(var(--i, 0) * 10%) cover calc(30% + var(--i, 0) * 6%);
			}
			.step {
				animation: reveal linear both;
				animation-timeline: view();
				animation-range: entry calc(var(--i) * 30%) cover calc(25% + var(--i) * 6%);
			}
			.steps-line {
				transform-origin: left;
				animation: grow linear both;
				animation-timeline: view();
				animation-range: entry 60% cover 50%;
			}
			.layer-view {
				animation: drift-view linear both;
				animation-timeline: view();
			}
			.cta {
				animation: zoom-in linear both;
				animation-timeline: view();
				animation-range: entry 0% cover 30%;
			}
		}
	}

	@keyframes word-in {
		from {
			opacity: 0;
			filter: blur(8px);
			transform: translateY(0.6em) rotate(4deg);
		}
	}
	@keyframes draw {
		from {
			transform: scaleX(0);
		}
	}
	@keyframes rise {
		from {
			opacity: 0;
			transform: translateY(32px);
		}
	}
	@keyframes slide-in {
		from {
			opacity: 0;
			transform: translateX(24px);
		}
	}
	@keyframes float {
		from {
			transform: translateY(-14px) rotate(-8deg);
		}
		to {
			transform: translateY(14px) rotate(8deg);
		}
	}
	@keyframes bob {
		from {
			transform: translateY(-6px) rotate(1deg);
		}
		to {
			transform: translateY(6px) rotate(-1deg);
		}
	}
	@keyframes pulse {
		50% {
			transform: scale(1.4);
			opacity: 0.6;
		}
	}
	@keyframes cue {
		from {
			transform: translateY(0);
			opacity: 1;
		}
		to {
			transform: translateY(12px);
			opacity: 0;
		}
	}
	@keyframes marquee {
		to {
			transform: translateX(-50%);
		}
	}
	@keyframes glow {
		to {
			transform: scale(1.25);
			opacity: 0.6;
		}
	}
	@keyframes grow {
		from {
			transform: scaleX(0);
		}
		to {
			transform: scaleX(1);
		}
	}
	@keyframes drift {
		to {
			transform: translateY(calc(var(--d) * -40vh));
		}
	}
	@keyframes hero-out {
		to {
			opacity: 0.2;
			transform: scale(0.92);
			filter: blur(4px);
		}
	}
	@keyframes untilt {
		from {
			transform: rotateX(14deg) rotateY(-14deg) rotateZ(2deg);
		}
	}
	@keyframes fade-out {
		to {
			opacity: 0;
		}
	}
	@keyframes reveal {
		from {
			opacity: 0;
			transform: translateY(48px) scale(0.96);
		}
	}
	@keyframes drift-view {
		from {
			transform: translateY(-20%);
		}
		to {
			transform: translateY(40%);
		}
	}
	@keyframes zoom-in {
		from {
			opacity: 0.3;
			transform: scale(0.85);
			border-radius: 4rem;
		}
	}
</style>
