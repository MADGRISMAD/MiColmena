<script lang="ts">
	import { resolve } from '$app/paths';
	import ArrowRightIcon from '@lucide/svelte/icons/arrow-right';
	import AtSignIcon from '@lucide/svelte/icons/at-sign';
	import BadgeDollarSignIcon from '@lucide/svelte/icons/badge-dollar-sign';
	import BellRingIcon from '@lucide/svelte/icons/bell-ring';
	import BookOpenIcon from '@lucide/svelte/icons/book-open';
	import ChartColumnIcon from '@lucide/svelte/icons/chart-column';
	import CheckIcon from '@lucide/svelte/icons/check';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import InboxIcon from '@lucide/svelte/icons/inbox';
	import LanguagesIcon from '@lucide/svelte/icons/languages';
	import LockIcon from '@lucide/svelte/icons/lock';
	import MessageSquareQuoteIcon from '@lucide/svelte/icons/message-square-quote';
	import RocketIcon from '@lucide/svelte/icons/rocket';
	import ShieldCheckIcon from '@lucide/svelte/icons/shield-check';
	import SmileIcon from '@lucide/svelte/icons/smile';
	import SmartphoneIcon from '@lucide/svelte/icons/smartphone';
	import TimerIcon from '@lucide/svelte/icons/timer';
	import UsersIcon from '@lucide/svelte/icons/users';
	import ZapIcon from '@lucide/svelte/icons/zap';
	import type { LeadPlan, TicketPriority, TicketStatus } from '#lib/api/types.js';
	import DemoForm from '#lib/components/demo-form.svelte';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import Logo from '#lib/components/logo.svelte';
	import PriorityBadge from '#lib/components/priority-badge.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { ANNUAL_MONTHS_PAID, planPrice, plans } from '#lib/pricing.js';
	import { isAuthenticated } from '#lib/stores/auth.js';
	import { cn } from '#lib/utils.js';

	// Por qué MiColmena: lo que la distingue para una empresa en México.
	const reasons = [
		{
			icon: BadgeDollarSignIcon,
			title: 'Precio en pesos',
			text: 'Pagas en MXN por agente. Sin cobros en dólares ni sorpresas por el tipo de cambio.'
		},
		{
			icon: LanguagesIcon,
			title: 'En español de verdad',
			text: 'Interfaz, correos y centro de ayuda en español. La búsqueda entiende «impresoras» cuando buscas «impresora».'
		},
		{
			icon: RocketIcon,
			title: 'Listo en una tarde',
			text: 'Das de alta a tu equipo, eliges categorías y plazos, y empiezas. Sin meses de implementación ni consultores.'
		}
	];

	// Funciones agrupadas por quién las usa. Todas existen hoy en el producto.
	const featureGroups = [
		{
			title: 'Para tus agentes',
			items: [
				{
					icon: InboxIcon,
					title: 'Bandeja con vistas',
					text: 'Lo tuyo, lo sin asignar, lo que espera al cliente y tus propias vistas guardadas.'
				},
				{
					icon: MessageSquareQuoteIcon,
					title: 'Respuestas guardadas',
					text: 'Plantillas con el nombre del cliente que además cambian el estado al enviar.'
				},
				{
					icon: AtSignIcon,
					title: 'Notas internas y menciones',
					text: 'Coordínate dentro del ticket con @compañero sin que el cliente lo vea.'
				},
				{
					icon: ZapIcon,
					title: 'Acciones masivas',
					text: 'Asigna, prioriza, etiqueta o cierra decenas de tickets de un clic.'
				}
			]
		},
		{
			title: 'Para tus clientes',
			items: [
				{
					icon: SmartphoneIcon,
					title: 'Portal de soporte',
					text: 'Abren tickets, adjuntan capturas y siguen la conversación desde la computadora o el celular.'
				},
				{
					icon: BookOpenIcon,
					title: 'Centro de ayuda',
					text: 'Artículos públicos que se sugieren mientras escriben, antes de abrir un ticket.'
				},
				{
					icon: BellRingIcon,
					title: 'Avisos al momento',
					text: 'Correo y notificaciones en tiempo real cuando les responden o se resuelve su caso.'
				},
				{
					icon: SmileIcon,
					title: 'Encuesta de satisfacción',
					text: 'Al resolver, califican la atención con un clic.'
				}
			]
		},
		{
			title: 'Para quien dirige',
			items: [
				{
					icon: TimerIcon,
					title: 'SLA por prioridad',
					text: 'Plazos de primera respuesta y resolución, con aviso antes de que venzan.'
				},
				{
					icon: ChartColumnIcon,
					title: 'Reportes y CSV',
					text: 'Volumen, tiempos, cumplimiento y satisfacción por agente y categoría.'
				},
				{
					icon: UsersIcon,
					title: 'Roles y equipo',
					text: 'Administradores, agentes y clientes; da de alta o desactiva cuentas en segundos.'
				},
				{
					icon: ShieldCheckIcon,
					title: 'Seguro desde el inicio',
					text: 'Contraseñas cifradas, bloqueo ante intentos repetidos y sesiones que se cierran al cambiar la contraseña.'
				}
			]
		}
	];

	const steps = [
		{
			title: 'Pide tu demo',
			text: 'Te mostramos MiColmena con casos como los tuyos y resolvemos tus dudas.'
		},
		{
			title: 'Configura tu equipo',
			text: 'Das de alta a tus agentes, tus categorías y los plazos de atención.'
		},
		{
			title: 'Recibe tickets',
			text: 'Tus clientes escriben desde tu portal y tu equipo los resuelve en orden.'
		}
	];

	const faqs = [
		{
			q: '¿Los precios son en pesos?',
			a: 'Sí. Todos los precios están en pesos mexicanos (MXN) por agente al mes, más IVA. No dependen del tipo de cambio.'
		},
		{
			q: '¿Cobran por los clientes que abren tickets?',
			a: 'No. Solo pagas por los agentes, las personas de tu equipo que atienden tickets. Tus clientes y el centro de ayuda no cuentan.'
		},
		{
			q: '¿Necesito instalar algo?',
			a: 'No. MiColmena funciona en el navegador, en la computadora o en el celular, para tu equipo y para tus clientes.'
		},
		{
			q: '¿Puedo verlo antes de contratar?',
			a: 'Sí. Llena el formulario de demo y te enseñamos MiColmena con casos parecidos a los de tu empresa.'
		},
		{
			q: '¿Mis clientes necesitan crear una cuenta?',
			a: 'Para abrir y seguir sus tickets, sí: se registran con su email en un minuto. El centro de ayuda se puede leer sin cuenta.'
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
			title: 'No llega la factura del pedido 4521',
			status: 'open',
			priority: 'urgent',
			who: 'Marta Soto'
		},
		{
			id: 127,
			title: 'La terminal no imprime el ticket de venta',
			status: 'in_progress',
			priority: 'high',
			who: 'Luis Ramírez'
		},
		{
			id: 126,
			title: 'Cambio de datos fiscales',
			status: 'waiting',
			priority: 'medium',
			who: 'Marta Soto'
		},
		{
			id: 125,
			title: 'Alta de usuario para la sucursal Monterrey',
			status: 'resolved',
			priority: 'low',
			who: 'Luis Ramírez'
		}
	];

	const headline = 'Mesa de ayuda profesional, con precio'.split(' ');

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
		'Precios en pesos',
		'Bandeja compartida',
		'SLA por prioridad',
		'Respuestas guardadas',
		'Centro de ayuda',
		'Reportes en CSV',
		'Tiempo real',
		'Encuestas de satisfacción'
	];

	let annual = $state(true);
	let demoPlan = $state<LeadPlan | ''>('');
	let openFaq = $state<number | null>(0);

	function choosePlan(id: LeadPlan) {
		demoPlan = id;
		document.getElementById('demo')?.scrollIntoView({ behavior: 'smooth' });
		setTimeout(() => document.getElementById('lead-name')?.focus({ preventScroll: true }), 500);
	}

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
	<title>MiColmena · Mesa de ayuda en español con precios en pesos</title>
	<meta
		name="description"
		content="Mesa de ayuda para empresas en México: tickets, SLA, centro de ayuda y reportes en español, con precios en pesos por agente. Solicita una demo."
	/>
	<meta property="og:title" content="MiColmena · Mesa de ayuda en español con precios en pesos" />
	<meta
		property="og:description"
		content="Tickets, SLA, centro de ayuda y reportes para tu equipo de soporte, en español y en pesos."
	/>
	<meta property="og:type" content="website" />
	<meta property="og:locale" content="es_MX" />
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
				<a href="#por-que" class="hover:text-foreground">Por qué MiColmena</a>
				<a href="#funciones" class="hover:text-foreground">Funciones</a>
				<a href="#precios" class="hover:text-foreground">Precios</a>
				<a href="#preguntas" class="hover:text-foreground">Preguntas</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
				<a href={resolve('/help')} class="hover:text-foreground">Ayuda</a>
			</nav>
			<div class="flex items-center gap-2">
				{#if $isAuthenticated}
					<Button href={resolve('/(app)/tickets')}>Ir a mis tickets</Button>
				{:else}
					<Button variant="ghost" href={resolve('/login')}>Iniciar sesión</Button>
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- ancla dentro de esta página -->
					<Button href="#demo" class="hidden sm:inline-flex">Solicitar demo</Button>
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
				class="hero-inner relative mx-auto grid w-full max-w-6xl grid-cols-[minmax(0,1fr)] items-center gap-12 px-4 py-16 sm:py-24 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)] lg:py-0"
			>
				<div class="layer" style="--d: 0.2">
					<p
						class="enter inline-flex items-center gap-2 rounded-full border bg-card/80 px-3 py-1 text-xs font-medium text-muted-foreground shadow-xs backdrop-blur"
					>
						<span class="hex pulse size-2.5 bg-honey" aria-hidden="true"></span>
						Hecha para empresas en México
					</p>
					<h1 class="mt-5 text-4xl font-bold tracking-tight text-balance sm:text-5xl xl:text-6xl">
						{#each headline as word, i (i)}
							<span class="word mr-[0.25em]" style="--i: {i}">{word}</span><wbr />
						{/each}
						<span
							class="word relative whitespace-nowrap text-primary dark:text-honey"
							style="--i: {headline.length}"
						>
							en pesos
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
						Tickets, SLA, centro de ayuda y reportes en una sola herramienta en español. Lo que
						esperas de una plataforma de soporte empresarial, con un precio pensado para México.
					</p>
					<div class="enter mt-8 flex flex-wrap gap-3" style="--delay: 850ms">
						<!-- eslint-disable svelte/no-navigation-without-resolve -- anclas dentro de esta página -->
						<Button size="lg" href="#demo">
							Solicitar una demo
							<ArrowRightIcon aria-hidden="true" />
						</Button>
						<Button size="lg" variant="outline" href="#precios">Ver precios</Button>
						<!-- eslint-enable svelte/no-navigation-without-resolve -->
					</div>
					<ul
						class="enter mt-8 flex flex-wrap gap-x-5 gap-y-2 text-sm text-muted-foreground"
						style="--delay: 1000ms"
					>
						{#each [`Desde ${planPrice(plans[0], false)} MXN por agente`, 'Sin pagar en dólares', 'Todo en español'] as point (point)}
							<li class="flex items-center gap-1.5">
								<CheckIcon class="size-4 text-amber-600" aria-hidden="true" />{point}
							</li>
						{/each}
					</ul>
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
							<p class="mt-1">@Marta revisa si el CFDI se timbró antes de reenviarlo.</p>
						</div>
					</div>
				</div>
			</div>

			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- ancla dentro de esta página -->
			<a
				href="#por-que"
				class="scroll-cue absolute bottom-6 left-1/2 hidden -translate-x-1/2 lg:block"
				aria-label="Bajar a por qué MiColmena"
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

			<!-- Por qué MiColmena -->
			<section id="por-que" class="mx-auto max-w-6xl scroll-mt-20 px-4 py-24">
				<div class="reveal max-w-2xl">
					<p class="text-sm font-semibold tracking-wide text-amber-700 uppercase dark:text-honey">
						Por qué MiColmena
					</p>
					<h2 class="mt-2 text-3xl font-bold tracking-tight text-balance sm:text-4xl">
						Soporte de nivel empresarial, sin precio de Silicon Valley
					</h2>
					<p class="mt-3 text-muted-foreground">
						Las grandes plataformas de soporte cobran lo mismo en México que en Estados Unidos, y en
						dólares. MiColmena te da lo que tu equipo usa todos los días, en tu idioma y en tu
						moneda.
					</p>
				</div>
				<div class="mt-12 grid gap-4 md:grid-cols-3">
					{#each reasons as reason, i (reason.title)}
						<div
							class="reveal group rounded-2xl border bg-card p-7 shadow-xs transition-[box-shadow,translate] duration-300 hover:-translate-y-1 hover:shadow-lg hover:shadow-amber-900/10"
							style="--i: {i}"
						>
							<span
								class="hex grid size-12 place-items-center bg-honey text-honey-foreground transition-transform duration-500 group-hover:scale-110 group-hover:rotate-[30deg]"
							>
								<reason.icon
									class="size-6 transition-transform duration-500 group-hover:-rotate-[30deg]"
									aria-hidden="true"
								/>
							</span>
							<h3 class="mt-5 text-lg font-semibold">{reason.title}</h3>
							<p class="mt-2 text-muted-foreground">{reason.text}</p>
						</div>
					{/each}
				</div>
			</section>

			<!-- Funciones -->
			<section id="funciones" class="scroll-mt-20 border-y bg-card/60 py-24">
				<div class="mx-auto max-w-6xl px-4">
					<div class="reveal max-w-2xl">
						<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">
							Todo tu soporte en una sola colmena
						</h2>
						<p class="mt-3 text-muted-foreground">
							Cada función está pensada para que tu equipo responda más rápido y tus clientes sepan
							siempre en qué va su caso.
						</p>
					</div>
					<div class="mt-12 grid gap-10 lg:grid-cols-3">
						{#each featureGroups as group, g (group.title)}
							<div class="reveal" style="--i: {g}">
								<h3 class="flex items-center gap-2 font-semibold">
									<span class="hex size-3 bg-honey" aria-hidden="true"></span>
									{group.title}
								</h3>
								<ul class="mt-5 grid gap-5">
									{#each group.items as item (item.title)}
										<li class="flex gap-3">
											<span
												class="grid size-9 shrink-0 place-items-center rounded-lg bg-honey/15 text-amber-700 dark:text-honey"
											>
												<item.icon class="size-4" aria-hidden="true" />
											</span>
											<span>
												<span class="block font-medium">{item.title}</span>
												<span class="mt-0.5 block text-sm text-muted-foreground">{item.text}</span>
											</span>
										</li>
									{/each}
								</ul>
							</div>
						{/each}
					</div>
				</div>
			</section>

			<!-- Cómo empezar -->
			<section
				id="como-funciona"
				class="bg-honeycomb-dark relative scroll-mt-20 overflow-hidden bg-sidebar text-white"
			>
				<div
					class="layer-view pointer-events-none absolute -top-40 -right-40 size-[32rem] rounded-full bg-honey/15 blur-3xl"
					aria-hidden="true"
				></div>
				<div class="relative mx-auto max-w-6xl px-4 py-24">
					<h2 class="reveal text-3xl font-bold tracking-tight sm:text-4xl">
						Empieza en tres pasos
					</h2>
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

			<!-- Precios -->
			<section id="precios" class="mx-auto max-w-6xl scroll-mt-20 px-4 py-24">
				<div class="reveal mx-auto max-w-2xl text-center">
					<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">Precios claros, en pesos</h2>
					<p class="mt-3 text-muted-foreground">
						Pagas por cada agente de tu equipo. Tus clientes no cuentan.
					</p>
					<div
						class="mt-8 inline-flex rounded-full border bg-muted p-1 text-sm"
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

				<div class="mt-12 grid items-start gap-6 lg:grid-cols-3">
					{#each plans as plan, i (plan.id)}
						{@const price = planPrice(plan, annual)}
						<div
							class={cn(
								'reveal relative flex h-full flex-col rounded-2xl border bg-card p-7 shadow-xs',
								plan.highlighted && 'border-2 border-honey shadow-xl shadow-amber-900/10 lg:-mt-4'
							)}
							style="--i: {i}"
						>
							{#if plan.highlighted}
								<span
									class="absolute -top-3 left-7 rounded-full bg-honey px-3 py-0.5 text-xs font-semibold text-honey-foreground"
								>
									Recomendado
								</span>
							{/if}
							<h3 class="text-lg font-semibold">{plan.name}</h3>
							<p class="mt-1 min-h-10 text-sm text-muted-foreground">{plan.description}</p>
							<div class="mt-5 flex items-end gap-1">
								{#if price}
									<span class="text-4xl font-bold tracking-tight tabular-nums">{price}</span>
									<span class="pb-1 text-sm text-muted-foreground">MXN / agente / mes</span>
								{:else}
									<span class="text-4xl font-bold tracking-tight">A la medida</span>
								{/if}
							</div>
							<p class="mt-1 text-xs text-muted-foreground">
								{plan.agents} · {price
									? annual
										? 'pago anual, más IVA'
										: 'pago mensual, más IVA'
									: 'te enviamos una cotización'}
							</p>
							<Button
								class="mt-6 w-full"
								variant={plan.highlighted ? 'default' : 'outline'}
								onclick={() => choosePlan(plan.id)}
							>
								{price ? 'Solicitar demo' : 'Cotizar'}
							</Button>
							<ul class="mt-6 grid gap-2.5 text-sm">
								{#each plan.features as feature (feature)}
									<li class="flex gap-2">
										<CheckIcon class="mt-0.5 size-4 shrink-0 text-amber-600" aria-hidden="true" />
										{feature}
									</li>
								{/each}
							</ul>
						</div>
					{/each}
				</div>
				<p class="mt-8 text-center text-sm text-muted-foreground">
					Precios en pesos mexicanos por agente al mes. IVA no incluido.
				</p>
			</section>

			<!-- Preguntas frecuentes -->
			<section id="preguntas" class="scroll-mt-20 border-t bg-card/60 py-24">
				<div class="mx-auto grid max-w-6xl gap-10 px-4 lg:grid-cols-[1fr_1.6fr]">
					<div class="reveal">
						<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">Preguntas frecuentes</h2>
						<p class="mt-3 text-muted-foreground">
							¿Tienes otra duda? Escríbela en el formulario de demo y te respondemos.
						</p>
					</div>
					<ul class="reveal divide-y rounded-2xl border bg-card" style="--i: 1">
						{#each faqs as faq, i (faq.q)}
							<li>
								<h3>
									<button
										type="button"
										class="flex w-full items-center justify-between gap-4 px-6 py-5 text-left font-medium"
										aria-expanded={openFaq === i}
										aria-controls={`faq-${i}`}
										onclick={() => (openFaq = openFaq === i ? null : i)}
									>
										{faq.q}
										<ChevronDownIcon
											class={cn(
												'size-4 shrink-0 text-muted-foreground transition-transform',
												openFaq === i && 'rotate-180'
											)}
											aria-hidden="true"
										/>
									</button>
								</h3>
								<div id={`faq-${i}`} hidden={openFaq !== i} class="px-6 pb-5 text-muted-foreground">
									{faq.a}
								</div>
							</li>
						{/each}
					</ul>
				</div>
			</section>

			<!-- Solicitar demo -->
			<section id="demo" class="mx-auto max-w-6xl scroll-mt-20 px-4 py-24">
				<div
					class="cta bg-honeycomb-dark relative grid gap-10 overflow-hidden rounded-3xl bg-sidebar p-8 sm:p-12 lg:grid-cols-[1fr_1.3fr]"
				>
					<div
						class="glow-pulse pointer-events-none absolute -bottom-32 -left-20 size-[30rem] rounded-full bg-honey/25 blur-3xl"
						aria-hidden="true"
					></div>
					<div class="relative text-white">
						<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">Pon orden en tu soporte</h2>
						<p class="mt-3 text-sidebar-foreground">
							Cuéntanos de tu equipo y te mostramos cómo MiColmena se adapta a tu forma de trabajar.
						</p>
						<ul class="mt-8 grid gap-3 text-sm">
							{#each ['Demo con casos parecidos a los tuyos', 'Te ayudamos a elegir el plan', 'Precios en pesos, sin letras chiquitas'] as point (point)}
								<li class="flex items-center gap-2">
									<span class="hex grid size-6 place-items-center bg-honey text-honey-foreground">
										<CheckIcon class="size-3.5" aria-hidden="true" />
									</span>
									{point}
								</li>
							{/each}
						</ul>
					</div>
					<div class="relative rounded-2xl bg-card p-6 text-card-foreground shadow-2xl sm:p-8">
						<DemoForm bind:plan={demoPlan} />
					</div>
				</div>
			</section>
		</div>
	</main>

	<footer class="relative z-10 border-t bg-background">
		<div
			class="mx-auto grid max-w-6xl gap-8 px-4 py-10 text-sm text-muted-foreground sm:grid-cols-[1fr_auto]"
		>
			<div>
				<Logo class="origin-left scale-90" />
				<p class="mt-3 max-w-xs">Mesa de ayuda en español para empresas en México.</p>
			</div>
			<nav class="flex flex-wrap gap-x-6 gap-y-2" aria-label="Pie de página">
				<!-- eslint-disable svelte/no-navigation-without-resolve -- anclas dentro de esta página -->
				<a href="#precios" class="hover:text-foreground">Precios</a>
				<a href="#demo" class="hover:text-foreground">Solicitar demo</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
				<a href={resolve('/help')} class="hover:text-foreground">Centro de ayuda</a>
				<a href={resolve('/login')} class="hover:text-foreground">Iniciar sesión</a>
			</nav>
			<p class="sm:col-span-2">
				© {new Date().getFullYear()} MiColmena · Precios en pesos mexicanos, IVA no incluido.
			</p>
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
