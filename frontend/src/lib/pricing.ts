// Precios de MiColmena. Para cambiarlos edita SOLO este archivo: la calculadora de la página
// principal, el formulario de demo y el panel de solicitudes leen de aquí.
//
// Precio mensual = cuota según el tamaño de la empresa + precio de cada agente (más barato por volumen).
// Así ni una empresa grande paga como una pequeña por tener pocos agentes, ni compartir una cuenta
// de agente sale tan a cuenta. Todos los precios en pesos, sin IVA.

/** Cuota mensual según cuántas personas tiene la empresa. null = se cotiza. */
export const COMPANY_SIZES: { upTo: number | null; fee: number | null }[] = [
	{ upTo: 10, fee: 99 },
	{ upTo: 25, fee: 199 },
	{ upTo: 50, fee: 399 },
	{ upTo: 100, fee: 699 },
	{ upTo: 250, fee: 1199 },
	{ upTo: 500, fee: 1999 },
	{ upTo: 1000, fee: 2999 },
	{ upTo: null, fee: null }
];

/** Precio mensual por agente, por tramos: los primeros 5 a $149, del 6 al 20 a $129, etc. */
export const AGENT_TIERS: { upTo: number; price: number }[] = [
	{ upTo: 5, price: 149 },
	{ upTo: 20, price: 129 },
	{ upTo: 50, price: 109 }
];

/** A partir de aquí la calculadora pide cotización. */
export const MAX_AGENTS = AGENT_TIERS[AGENT_TIERS.length - 1].upTo;

/** Con pago anual se cobran 10 meses: 2 meses gratis. */
export const ANNUAL_MONTHS_PAID = 10;

export interface Estimate {
	/** true si se sale de la calculadora y hay que cotizar. */
	custom: boolean;
	companyFee: number;
	/** Agentes agrupados por tramo de precio: [{ count: 5, price: 149 }, …]. */
	agentLines: { count: number; price: number }[];
	/** Total al mes con pago mensual. */
	monthly: number;
	/** Total al mes con pago anual (lo que cuesta cada mes al pagar el año). */
	monthlyAnnual: number;
	/** Total del año con pago anual. */
	annualTotal: number;
}

/** Calcula el precio para `agents` agentes en una empresa de `people` personas. */
export function estimate(agents: number, people: number): Estimate {
	const size = COMPANY_SIZES.find((s) => s.upTo === null || people <= s.upTo)!;
	const custom = size.fee === null || agents > MAX_AGENTS;
	const agentLines: Estimate['agentLines'] = [];
	let counted = 0;
	for (const tier of AGENT_TIERS) {
		const count = Math.min(agents, tier.upTo) - counted;
		if (count <= 0) break;
		agentLines.push({ count, price: tier.price });
		counted += count;
	}
	const companyFee = size.fee ?? 0;
	const monthly = companyFee + agentLines.reduce((sum, l) => sum + l.count * l.price, 0);
	const annualTotal = monthly * ANNUAL_MONTHS_PAID;
	return {
		custom,
		companyFee,
		agentLines,
		monthly,
		monthlyAnnual: Math.round(annualTotal / 12),
		annualTotal
	};
}

/** Puntos del slider de tamaño de empresa (el último significa "más de 1,000"). */
export const PEOPLE_STOPS = [5, 10, 25, 50, 100, 250, 500, 1000, 1001];

export function peopleLabel(people: number): string {
	return people > 1000 ? 'Más de 1,000' : `Hasta ${people.toLocaleString('es-MX')}`;
}

const mxn = new Intl.NumberFormat('es-MX', {
	style: 'currency',
	currency: 'MXN',
	maximumFractionDigits: 0
});

/** "$1,299". */
export function money(amount: number): string {
	return mxn.format(amount);
}

/** Precio más bajo posible, para "Desde $X al mes". */
export const STARTING_PRICE = estimate(1, PEOPLE_STOPS[0]).monthly;
