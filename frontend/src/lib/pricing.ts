// Planes y precios de la página principal. Para cambiar precios o lo que incluye cada plan,
// edita solo este archivo: la landing, el formulario de demo y el panel de solicitudes lo leen de aquí.
import type { LeadPlan } from '#lib/api/types.js';

export interface Plan {
	id: LeadPlan;
	name: string;
	/** Precio por agente al mes en pesos, sin IVA. null = se cotiza. */
	monthly: number | null;
	description: string;
	/** Límite de agentes que se muestra junto al precio. */
	agents: string;
	features: string[];
	highlighted?: boolean;
}

/** Con pago anual se cobran 10 meses: 2 meses gratis. */
export const ANNUAL_MONTHS_PAID = 10;

export const plans: Plan[] = [
	{
		id: 'emprendedor',
		name: 'Emprendedor',
		monthly: 149,
		description: 'Para empezar a ordenar el soporte de un equipo pequeño.',
		agents: 'Hasta 3 agentes',
		features: [
			'Tickets ilimitados',
			'Portal para que tus clientes abran y sigan sus tickets',
			'Notas internas, menciones y adjuntos',
			'Centro de ayuda público',
			'Avisos por correo y en tiempo real'
		]
	},
	{
		id: 'profesional',
		name: 'Profesional',
		monthly: 249,
		description: 'Para equipos que necesitan medir y cumplir tiempos de atención.',
		agents: 'Hasta 25 agentes',
		features: [
			'Todo lo de Emprendedor',
			'SLA por prioridad con avisos de vencimiento',
			'Respuestas guardadas y acciones masivas',
			'Reportes por agente y exportación a CSV',
			'Encuestas de satisfacción',
			'Vistas guardadas y etiquetas'
		],
		highlighted: true
	},
	{
		id: 'empresa',
		name: 'Empresa',
		monthly: null,
		description: 'Para operaciones grandes o con necesidades especiales.',
		agents: 'Más de 25 agentes',
		features: [
			'Todo lo de Profesional',
			'Precio por volumen',
			'Acompañamiento en la puesta en marcha',
			'Condiciones a la medida de tu empresa'
		]
	}
];

const mxn = new Intl.NumberFormat('es-MX', {
	style: 'currency',
	currency: 'MXN',
	maximumFractionDigits: 0
});

/** Precio mensual por agente según la forma de pago: "$249". */
export function planPrice(plan: Plan, annual: boolean): string | null {
	if (plan.monthly === null) return null;
	const perMonth = annual ? Math.round((plan.monthly * ANNUAL_MONTHS_PAID) / 12) : plan.monthly;
	return mxn.format(perMonth);
}
