<script lang="ts">
	import { toast } from 'svelte-sonner';
	import { api, ApiError } from '#lib/api/index.js';
	import HexAvatar from '#lib/components/hex-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { roleLabels } from '#lib/format.js';
	import { setSession, user } from '#lib/stores/auth.js';

	let name = $state($user?.name ?? '');
	let email = $state($user?.email ?? '');
	let currentForEmail = $state('');
	let profileErrors = $state<Record<string, string>>({});
	let savingProfile = $state(false);

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let passwordErrors = $state<Record<string, string>>({});
	let savingPassword = $state(false);

	const emailChanged = $derived(email.trim() !== ($user?.email ?? ''));

	async function saveProfile(event: SubmitEvent) {
		event.preventDefault();
		profileErrors = {};
		savingProfile = true;
		try {
			user.set(
				await api.updateMe({
					name: name.trim(),
					email: email.trim(),
					current_password: emailChanged ? currentForEmail : undefined
				})
			);
			currentForEmail = '';
			toast.success('Perfil actualizado');
		} catch (err) {
			if (err instanceof ApiError && Object.keys(err.fields).length) profileErrors = err.fields;
			else toast.error(err instanceof Error ? err.message : 'No se pudo guardar');
		} finally {
			savingProfile = false;
		}
	}

	async function toggleEmails(event: Event) {
		const checked = (event.currentTarget as HTMLInputElement).checked;
		try {
			user.set(await api.updateMe({ email_notifications: checked }));
			toast.success(checked ? 'Recibirás avisos por correo' : 'Ya no recibirás avisos por correo');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'No se pudo guardar');
		}
	}

	async function savePassword(event: SubmitEvent) {
		event.preventDefault();
		passwordErrors = {};
		if (newPassword.length < 8) {
			passwordErrors = { new_password: 'Debe tener al menos 8 caracteres' };
			return;
		}
		if (newPassword !== confirmPassword) {
			passwordErrors = { confirm: 'Las contraseñas no coinciden' };
			return;
		}
		savingPassword = true;
		try {
			const res = await api.changePassword(currentPassword, newPassword);
			// Cambiar la contraseña cierra las demás sesiones; esta sigue con el token nuevo.
			setSession(res.token, res.user);
			currentPassword = newPassword = confirmPassword = '';
			toast.success('Contraseña cambiada. Se cerraron tus otras sesiones.');
		} catch (err) {
			if (err instanceof ApiError && Object.keys(err.fields).length) passwordErrors = err.fields;
			else toast.error(err instanceof Error ? err.message : 'No se pudo cambiar');
		} finally {
			savingPassword = false;
		}
	}
</script>

<PageHeader eyebrow="Cuenta" title="Mi perfil" description="Tus datos, avisos y contraseña." />

{#if $user}
	<div class="grid max-w-3xl gap-6">
		<Card.Root>
			<Card.Content class="flex-row items-center gap-4">
				<HexAvatar name={$user.name} id={$user.id} size="lg" />
				<div>
					<p class="font-semibold">{$user.name}</p>
					<p class="text-sm text-muted-foreground">{roleLabels[$user.role]} · {$user.email}</p>
				</div>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title>Datos personales</Card.Title>
			</Card.Header>
			<Card.Content>
				<form class="grid gap-4" onsubmit={saveProfile} novalidate>
					<div class="grid gap-1.5">
						<Label for="name">Nombre</Label>
						<Input
							id="name"
							bind:value={name}
							autocomplete="name"
							aria-invalid={!!profileErrors.name}
						/>
						{#if profileErrors.name}<p class="text-sm text-destructive">
								{profileErrors.name}
							</p>{/if}
					</div>
					<div class="grid gap-1.5">
						<Label for="email">Email</Label>
						<Input
							id="email"
							type="email"
							bind:value={email}
							autocomplete="email"
							aria-invalid={!!profileErrors.email}
						/>
						{#if profileErrors.email}<p class="text-sm text-destructive">
								{profileErrors.email}
							</p>{/if}
					</div>
					{#if emailChanged}
						<div class="grid gap-1.5">
							<Label for="current-for-email">Contraseña actual (para cambiar el email)</Label>
							<Input
								id="current-for-email"
								type="password"
								bind:value={currentForEmail}
								autocomplete="current-password"
								aria-invalid={!!profileErrors.current_password}
							/>
							{#if profileErrors.current_password}
								<p class="text-sm text-destructive">{profileErrors.current_password}</p>
							{/if}
						</div>
					{/if}
					<div>
						<Button type="submit" disabled={savingProfile}>
							{savingProfile ? 'Guardando…' : 'Guardar cambios'}
						</Button>
					</div>
				</form>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title>Avisos</Card.Title>
				<Card.Description>
					Las notificaciones siempre aparecen en la campana. Aquí decides si también te llegan por
					correo.
				</Card.Description>
			</Card.Header>
			<Card.Content>
				<label class="flex items-center gap-3 text-sm">
					<input
						type="checkbox"
						class="size-4 accent-amber-500"
						checked={$user.email_notifications}
						onchange={toggleEmails}
					/>
					Recibir avisos por correo
				</label>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Header>
				<Card.Title>Contraseña</Card.Title>
				<Card.Description
					>Al cambiarla se cierran tus sesiones en otros dispositivos.</Card.Description
				>
			</Card.Header>
			<Card.Content>
				<form class="grid gap-4" onsubmit={savePassword} novalidate>
					<div class="grid gap-1.5">
						<Label for="current-password">Contraseña actual</Label>
						<Input
							id="current-password"
							type="password"
							bind:value={currentPassword}
							autocomplete="current-password"
							aria-invalid={!!passwordErrors.current_password}
						/>
						{#if passwordErrors.current_password}
							<p class="text-sm text-destructive">{passwordErrors.current_password}</p>
						{/if}
					</div>
					<div class="grid gap-4 sm:grid-cols-2">
						<div class="grid gap-1.5">
							<Label for="new-password">Nueva contraseña</Label>
							<Input
								id="new-password"
								type="password"
								bind:value={newPassword}
								autocomplete="new-password"
								aria-invalid={!!passwordErrors.new_password}
							/>
							{#if passwordErrors.new_password}
								<p class="text-sm text-destructive">{passwordErrors.new_password}</p>
							{/if}
						</div>
						<div class="grid gap-1.5">
							<Label for="confirm-password">Repite la nueva contraseña</Label>
							<Input
								id="confirm-password"
								type="password"
								bind:value={confirmPassword}
								autocomplete="new-password"
								aria-invalid={!!passwordErrors.confirm}
							/>
							{#if passwordErrors.confirm}
								<p class="text-sm text-destructive">{passwordErrors.confirm}</p>
							{/if}
						</div>
					</div>
					<div>
						<Button type="submit" disabled={savingPassword}>
							{savingPassword ? 'Cambiando…' : 'Cambiar contraseña'}
						</Button>
					</div>
				</form>
			</Card.Content>
		</Card.Root>
	</div>
{/if}
