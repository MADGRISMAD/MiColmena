<script lang="ts">
	import { resolve } from '$app/paths';
	import ChartColumnIcon from '@lucide/svelte/icons/chart-column';
	import LockIcon from '@lucide/svelte/icons/lock';
	import SearchIcon from '@lucide/svelte/icons/search';
	import TicketIcon from '@lucide/svelte/icons/ticket';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { isAuthenticated } from '#lib/stores/auth.js';

	const features = [
		{
			icon: TicketIcon,
			title: 'Tickets en un solo lugar',
			text: 'Tus clientes abren tickets y tu equipo los asigna, prioriza y resuelve sin perder el hilo.'
		},
		{
			icon: LockIcon,
			title: 'Notas internas',
			text: 'Los agentes comentan entre ellos dentro del ticket sin que el cliente lo vea.'
		},
		{
			icon: SearchIcon,
			title: 'Búsqueda rápida',
			text: 'Encuentra cualquier ticket por texto, estado, prioridad o agente asignado.'
		},
		{
			icon: ChartColumnIcon,
			title: 'Dashboard',
			text: 'Tickets abiertos, sin asignar y tiempo medio de resolución, siempre a la vista.'
		}
	];
</script>

<div class="flex min-h-screen flex-col">
	<header class="border-b">
		<div class="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
			<a href={resolve('/')} class="flex items-center gap-2 font-semibold">
				<img src="/logo.png" alt="" class="size-8" />
				MiColmena
			</a>
			<nav class="flex items-center gap-2">
				{#if $isAuthenticated}
					<Button href={resolve('/(app)/tickets')}>Ir a mis tickets</Button>
				{:else}
					<Button variant="ghost" href={resolve('/login')}>Iniciar sesión</Button>
					<Button href={resolve('/register')}>Crear cuenta</Button>
				{/if}
			</nav>
		</div>
	</header>

	<main class="flex-1">
		<section class="mx-auto max-w-3xl px-4 py-20 text-center sm:py-28">
			<h1 class="text-4xl font-bold tracking-tight text-balance sm:text-5xl">
				El soporte de tu equipo, organizado como una colmena
			</h1>
			<p class="mt-6 text-lg text-pretty text-muted-foreground">
				MiColmena es una plataforma para gestionar tickets de soporte y atención al cliente. Recibe
				solicitudes, asígnalas y resuélvelas con todo el historial en un mismo lugar.
			</p>
			<div class="mt-10 flex flex-wrap justify-center gap-3">
				<Button size="lg" href={resolve('/register')}>Empezar gratis</Button>
				<Button size="lg" variant="outline" href={resolve('/login')}>Ya tengo cuenta</Button>
			</div>
		</section>

		<section class="mx-auto grid max-w-6xl gap-4 px-4 pb-20 sm:grid-cols-2 lg:grid-cols-4">
			{#each features as feature (feature.title)}
				<Card.Root>
					<Card.Header>
						<feature.icon class="mb-2 size-6 text-primary" aria-hidden="true" />
						<Card.Title>{feature.title}</Card.Title>
						<Card.Description>{feature.text}</Card.Description>
					</Card.Header>
				</Card.Root>
			{/each}
		</section>
	</main>

	<footer class="border-t py-6 text-center text-sm text-muted-foreground">
		© {new Date().getFullYear()} MiColmena
	</footer>
</div>
