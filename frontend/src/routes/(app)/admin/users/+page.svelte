<script lang="ts">
	import KeyRoundIcon from '@lucide/svelte/icons/key-round';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import SearchIcon from '@lucide/svelte/icons/search';
	import { toast } from 'svelte-sonner';
	import {
		api,
		ApiError,
		ROLES,
		type CreateUserInput,
		type Role,
		type User
	} from '#lib/api/index.js';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { NativeSelect, NativeSelectOption } from '#lib/components/ui/native-select/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatDateTime, roleLabels } from '#lib/format.js';
	import { user as me } from '#lib/stores/auth.js';
	import { loadOrg, org } from '#lib/stores/org.js';
	import { resolve } from '$app/paths';
	import { cn } from '#lib/utils.js';

	let users = $state<User[] | null>(null);
	let search = $state('');
	let roleFilter = $state<Role | ''>('');
	let showInactive = $state(false);
	let creating = $state(false);
	let draft = $state<CreateUserInput>({ name: '', email: '', role: 'agent', password: '' });
	let fieldErrors = $state<Record<string, string>>({});
	let busy = $state<number | null>(null);

	let requestId = 0;
	async function load() {
		const id = ++requestId;
		const res = await api
			.listUsers(roleFilter || undefined, { q: search.trim() || undefined, all: showInactive })
			.catch(() => ({ items: [] }));
		if (id === requestId) users = res.items;
	}

	$effect(() => {
		void roleFilter;
		void showInactive;
		void search;
		const t = setTimeout(load, 200);
		return () => clearTimeout(t);
	});

	function randomPassword() {
		const chars = 'abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789';
		const bytes = crypto.getRandomValues(new Uint8Array(14));
		return Array.from(bytes, (b) => chars[b % chars.length]).join('');
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		fieldErrors = {};
		try {
			const created = await api.createUser({
				...draft,
				name: draft.name.trim(),
				email: draft.email.trim()
			});
			toast.success(`${created.name} ya puede entrar con su email y la contraseña inicial`);
			creating = false;
			draft = { name: '', email: '', role: 'agent', password: '' };
			await Promise.all([load(), loadOrg()]);
		} catch (err) {
			if (err instanceof ApiError && Object.keys(err.fields).length) fieldErrors = err.fields;
			else toast.error(err instanceof Error ? err.message : 'No se pudo crear el usuario');
		}
	}

	async function change(u: User, input: Parameters<typeof api.updateUser>[1], message: string) {
		busy = u.id;
		try {
			const updated = await api.updateUser(u.id, input);
			users = users?.map((x) => (x.id === u.id ? updated : x)) ?? null;
			toast.success(message);
			loadOrg();
		} catch (err) {
			const fields = err instanceof ApiError ? Object.values(err.fields) : [];
			toast.error(fields[0] ?? (err instanceof Error ? err.message : 'No se pudo guardar'));
		} finally {
			busy = null;
		}
	}

	function resetPassword(u: User) {
		const password = prompt(
			`Nueva contraseña para ${u.name} (mínimo 8 caracteres). Sus sesiones abiertas se cerrarán.`,
			randomPassword()
		);
		if (password) change(u, { password }, 'Contraseña cambiada. Compártela de forma segura.');
	}

	function toggleActive(u: User) {
		if (
			u.active &&
			!confirm(
				`¿Desactivar a ${u.name}? No podrá entrar y sus tickets pendientes quedarán sin asignar.`
			)
		)
			return;
		change(u, { active: !u.active }, u.active ? 'Cuenta desactivada' : 'Cuenta reactivada');
	}
</script>

<PageHeader
	eyebrow="Administración"
	title="Usuarios"
	description="Da de alta agentes, cambia roles y desactiva cuentas."
>
	{#snippet actions()}
		<Button
			onclick={() => {
				creating = !creating;
				draft.password = randomPassword();
			}}
		>
			<PlusIcon aria-hidden="true" />
			Nuevo usuario
		</Button>
	{/snippet}
</PageHeader>

{#if $org && $org.max_agents !== null}
	<p
		class={cn(
			'mb-4 flex flex-wrap items-center gap-x-2 rounded-lg border px-4 py-3 text-sm',
			$org.agents >= $org.max_agents
				? 'border-amber-400 bg-amber-50 dark:bg-amber-500/10'
				: 'bg-card'
		)}
		data-testid="agent-limit"
	>
		<span>
			Agentes: <strong>{$org.agents} de {$org.max_agents}</strong> incluidos en tu plan.
		</span>
		{#if $org.agents >= $org.max_agents}
			<a
				href={resolve('/(app)/admin/plan')}
				class="font-medium text-primary underline dark:text-honey">Ampliar plan</a
			>
		{/if}
	</p>
{/if}

{#if creating}
	<Card.Root class="mb-6 border-honey/60">
		<Card.Header>
			<Card.Title>Nuevo usuario</Card.Title>
			<Card.Description>
				Comparte la contraseña inicial por un canal seguro; la persona puede cambiarla en su perfil.
			</Card.Description>
		</Card.Header>
		<Card.Content>
			<form class="grid gap-4 sm:grid-cols-2" onsubmit={create} novalidate>
				<div class="grid gap-1.5">
					<Label for="new-name">Nombre</Label>
					<Input id="new-name" bind:value={draft.name} aria-invalid={!!fieldErrors.name} />
					{#if fieldErrors.name}<p class="text-sm text-destructive">{fieldErrors.name}</p>{/if}
				</div>
				<div class="grid gap-1.5">
					<Label for="new-email">Email</Label>
					<Input
						id="new-email"
						type="email"
						bind:value={draft.email}
						aria-invalid={!!fieldErrors.email}
					/>
					{#if fieldErrors.email}<p class="text-sm text-destructive">{fieldErrors.email}</p>{/if}
				</div>
				<div class="grid gap-1.5">
					<Label for="new-role">Rol</Label>
					<NativeSelect
						id="new-role"
						class="w-full"
						bind:value={draft.role}
						aria-invalid={!!fieldErrors.role}
					>
						{#each ROLES as role (role)}
							<NativeSelectOption value={role}>{roleLabels[role]}</NativeSelectOption>
						{/each}
					</NativeSelect>
					{#if fieldErrors.role}<p class="text-sm text-destructive">{fieldErrors.role}</p>{/if}
				</div>
				<div class="grid gap-1.5">
					<Label for="new-password">Contraseña inicial</Label>
					<div class="flex gap-2">
						<Input
							id="new-password"
							bind:value={draft.password}
							class="font-mono"
							aria-invalid={!!fieldErrors.password}
						/>
						<Button
							type="button"
							variant="outline"
							onclick={() => (draft.password = randomPassword())}>Generar</Button
						>
					</div>
					{#if fieldErrors.password}<p class="text-sm text-destructive">
							{fieldErrors.password}
						</p>{/if}
				</div>
				<div class="flex gap-2 sm:col-span-2">
					<Button type="submit">Crear usuario</Button>
					<Button type="button" variant="ghost" onclick={() => (creating = false)}>Cancelar</Button>
				</div>
			</form>
		</Card.Content>
	</Card.Root>
{/if}

<div class="mb-4 flex flex-wrap items-center gap-3">
	<div class="relative max-w-xs flex-1">
		<SearchIcon
			class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
			aria-hidden="true"
		/>
		<Input
			bind:value={search}
			placeholder="Buscar por nombre o email"
			class="pl-9"
			aria-label="Buscar usuarios"
		/>
	</div>
	<NativeSelect size="sm" bind:value={roleFilter} aria-label="Filtrar por rol">
		<NativeSelectOption value="">Todos los roles</NativeSelectOption>
		{#each ROLES as role (role)}
			<NativeSelectOption value={role}>{roleLabels[role]}</NativeSelectOption>
		{/each}
	</NativeSelect>
	<label class="flex items-center gap-2 text-sm">
		<input type="checkbox" class="size-4 accent-amber-500" bind:checked={showInactive} />
		Mostrar desactivados
	</label>
</div>

<Card.Root class="gap-0 overflow-hidden py-0">
	{#if users === null}
		<div class="grid gap-3 p-4"><Skeleton class="h-10" /><Skeleton class="h-10" /></div>
	{:else if users.length === 0}
		<p class="p-8 text-center text-sm text-muted-foreground">Ningún usuario coincide.</p>
	{:else}
		<ul class="divide-y">
			{#each users as u (u.id)}
				<li class={cn('flex flex-wrap items-center gap-3 px-4 py-3', !u.active && 'opacity-60')}>
					<HexAvatar name={u.name} id={u.id} size="sm" />
					<div class="min-w-0 flex-1">
						<p class="truncate font-medium">
							{u.name}
							{#if u.id === $me?.id}<span class="text-xs font-normal text-muted-foreground"
									>(tú)</span
								>{/if}
							{#if !u.active}
								<span class="ml-1 rounded bg-muted px-1.5 text-xs font-normal">Desactivado</span>
							{/if}
							{#if u.permanent}
								<span
									class="ml-1 rounded bg-honey/25 px-1.5 text-xs font-normal"
									title="No se puede desactivar ni cambiar de rol">Permanente</span
								>
							{/if}
						</p>
						<p
							class="truncate text-xs text-muted-foreground"
							title={`Alta: ${formatDateTime(u.created_at)}`}
						>
							{u.email}
						</p>
					</div>
					<NativeSelect
						size="sm"
						value={u.role}
						disabled={busy === u.id || u.id === $me?.id || u.permanent}
						aria-label={`Rol de ${u.name}`}
						onchange={(e) => {
							const role = e.currentTarget.value as Role;
							change(u, { role }, `${u.name} ahora es ${roleLabels[role].toLowerCase()}`);
						}}
					>
						{#each ROLES as role (role)}
							<NativeSelectOption value={role}>{roleLabels[role]}</NativeSelectOption>
						{/each}
					</NativeSelect>
					<!-- La contraseña de un permanente solo la cambia otro permanente. -->
					<Button
						variant="ghost"
						size="icon-sm"
						disabled={busy === u.id || (u.permanent && u.id !== $me?.id && !$me?.permanent)}
						onclick={() => resetPassword(u)}
						aria-label={`Cambiar la contraseña de ${u.name}`}
						title="Cambiar contraseña"
					>
						<KeyRoundIcon aria-hidden="true" />
					</Button>
					{#if u.id !== $me?.id && !u.permanent}
						<Button
							variant="outline"
							size="sm"
							disabled={busy === u.id}
							onclick={() => toggleActive(u)}
						>
							{u.active ? 'Desactivar' : 'Reactivar'}
						</Button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</Card.Root>
