import ChartIcon from '@lucide/svelte/icons/chart-column';
import KeyIcon from '@lucide/svelte/icons/key';
import LaptopIcon from '@lucide/svelte/icons/laptop';
import MailIcon from '@lucide/svelte/icons/mail';
import PackageIcon from '@lucide/svelte/icons/package';
import PhoneIcon from '@lucide/svelte/icons/phone';
import ShieldIcon from '@lucide/svelte/icons/shield';
import UsersIcon from '@lucide/svelte/icons/users';
import WrenchIcon from '@lucide/svelte/icons/wrench';
import type { CatalogIcon } from '#lib/api/index.js';

export const catalogIcons = {
	package: PackageIcon,
	shield: ShieldIcon,
	mail: MailIcon,
	phone: PhoneIcon,
	users: UsersIcon,
	wrench: WrenchIcon,
	laptop: LaptopIcon,
	key: KeyIcon,
	chart: ChartIcon
} satisfies Record<CatalogIcon, unknown>;

export const catalogIconLabels: Record<CatalogIcon, string> = {
	package: 'Paquete',
	shield: 'Seguridad',
	mail: 'Correo',
	phone: 'Teléfono',
	users: 'Personas',
	wrench: 'Herramienta',
	laptop: 'Equipo',
	key: 'Llave',
	chart: 'Gráfica'
};
