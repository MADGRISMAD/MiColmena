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
</script>

<svelte:head>
	<title>MiColmena · Mesa de ayuda para tu equipo de soporte</title>
	<meta
		name="description"
		content="MiColmena organiza los tickets de soporte de tu equipo: bandeja compartida, notas internas, prioridades y dashboard."
	/>
</svelte:head>

<div class="flex min-h-screen flex-col">
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
	</header>

	<main class="flex-1">
		<!-- Hero -->
		<section class="bg-honeycomb relative overflow-hidden border-b">
			<div
				class="pointer-events-none absolute -top-32 left-1/2 size-[36rem] -translate-x-1/2 rounded-full bg-honey/25 blur-3xl"
				aria-hidden="true"
			></div>
			<div
				class="relative mx-auto grid max-w-6xl items-center gap-12 px-4 py-16 sm:py-24 lg:grid-cols-[1fr_1.1fr]"
			>
				<div>
					<p
						class="inline-flex items-center gap-2 rounded-full border bg-card/80 px-3 py-1 text-xs font-medium text-muted-foreground shadow-xs"
					>
						<span class="hex size-2.5 bg-honey" aria-hidden="true"></span>
						Mesa de ayuda para equipos de soporte
					</p>
					<h1 class="mt-5 text-4xl font-bold tracking-tight text-balance sm:text-5xl">
						El soporte de tu equipo, organizado como una
						<span class="relative whitespace-nowrap text-primary dark:text-honey">
							colmena
							<span
								class="absolute inset-x-0 -bottom-1 h-2 rounded-full bg-honey/50"
								aria-hidden="true"
							></span>
						</span>
					</h1>
					<p class="mt-6 max-w-xl text-lg text-pretty text-muted-foreground">
						Recibe las solicitudes de tus clientes, repártelas en el equipo y resuélvelas con todo
						el historial en un mismo lugar. Sencillo de usar desde el primer día.
					</p>
					<div class="mt-8 flex flex-wrap gap-3">
						<Button size="lg" href={resolve('/register')}>
							Crear una cuenta
							<ArrowRightIcon aria-hidden="true" />
						</Button>
						<Button size="lg" variant="outline" href={resolve('/login')}>Ya tengo cuenta</Button>
					</div>
				</div>

				<!-- Vista previa del producto, hecha con los mismos componentes de la app -->
				<div class="relative mb-16 lg:mb-0" aria-hidden="true">
					<div class="overflow-hidden rounded-xl border bg-card shadow-2xl shadow-amber-900/10">
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
								{#each preview as row (row.id)}
									<li class="flex items-center gap-3 px-4 py-3">
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
					<div
						class="absolute -right-3 -bottom-20 hidden w-64 rotate-1 rounded-xl border border-dashed border-amber-400 bg-amber-50 p-3 text-sm shadow-lg sm:block dark:bg-amber-950"
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
		</section>

		<!-- Funciones -->
		<section id="funciones" class="mx-auto max-w-6xl scroll-mt-20 px-4 py-20">
			<div class="max-w-2xl">
				<h2 class="text-3xl font-bold tracking-tight">
					Todo lo que tu equipo necesita, nada que estorbe
				</h2>
				<p class="mt-3 text-muted-foreground">
					Las herramientas de soporte grandes hacen de todo; MiColmena se enfoca en que atender a
					tus clientes sea rápido y ordenado.
				</p>
			</div>
			<div class="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each features as feature (feature.title)}
					<div class="rounded-xl border bg-card p-6 shadow-xs transition-shadow hover:shadow-md">
						<span class="hex grid size-11 place-items-center bg-honey text-honey-foreground">
							<feature.icon class="size-5" aria-hidden="true" />
						</span>
						<h3 class="mt-4 font-semibold">{feature.title}</h3>
						<p class="mt-2 text-sm text-muted-foreground">{feature.text}</p>
					</div>
				{/each}
			</div>
		</section>

		<!-- Cómo funciona -->
		<section id="como-funciona" class="scroll-mt-20 border-y bg-card">
			<div class="mx-auto max-w-6xl px-4 py-20">
				<h2 class="text-3xl font-bold tracking-tight">Cómo funciona</h2>
				<ol class="mt-12 grid gap-10 md:grid-cols-3">
					{#each steps as step, i (step.title)}
						<li>
							<span
								class="hex grid size-12 place-items-center bg-sidebar text-lg font-bold text-honey"
							>
								{i + 1}
							</span>
							<h3 class="mt-4 font-semibold">{step.title}</h3>
							<p class="mt-2 text-sm text-muted-foreground">{step.text}</p>
						</li>
					{/each}
				</ol>
			</div>
		</section>

		<!-- Llamada final -->
		<section class="mx-auto max-w-6xl px-4 py-20">
			<div
				class="bg-honeycomb-dark relative overflow-hidden rounded-2xl bg-sidebar px-8 py-14 text-center"
			>
				<div
					class="pointer-events-none absolute -bottom-32 left-1/2 size-[30rem] -translate-x-1/2 rounded-full bg-honey/20 blur-3xl"
					aria-hidden="true"
				></div>
				<h2 class="relative text-3xl font-bold tracking-tight text-white">
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
	</main>

	<footer class="border-t">
		<div
			class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 px-4 py-8 text-sm text-muted-foreground"
		>
			<Logo class="scale-90" />
			<p>© {new Date().getFullYear()} MiColmena</p>
		</div>
	</footer>
</div>
