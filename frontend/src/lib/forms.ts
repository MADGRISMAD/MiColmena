import { setError, setMessage, superForm } from 'sveltekit-superforms';
import { zod4Client, type ZodValidationSchema } from 'sveltekit-superforms/adapters';
import type { z } from 'zod/mini';
import { ApiError } from '#lib/api/index.js';

/**
 * Crea un formulario de Superforms en modo SPA: valida con Zod en el navegador y, si es válido,
 * llama a `submit`. Los errores por campo que devuelve la API (422) se muestran en su campo;
 * cualquier otro error se muestra como mensaje general del formulario ($message).
 *
 * Se pasan los valores iniciales explícitos y el adaptador solo-cliente (zod4Client) para no
 * incluir en el bundle el generador de JSON Schema que Superforms usa en el servidor.
 *
 * `submit` puede devolver datos con los que reiniciar el formulario (por ejemplo, vaciarlo tras enviar).
 */
export function apiForm<S extends ZodValidationSchema>(
	schema: S,
	initial: z.infer<S>,
	submit: (data: z.infer<S>) => Promise<Partial<z.infer<S>> | void>,
	options: { id?: string } = {}
) {
	return superForm(structuredClone(initial), {
		id: options.id,
		SPA: true,
		validators: zod4Client(schema),
		resetForm: false,
		taintedMessage: null,
		async onUpdate({ form }) {
			if (!form.valid) return;
			try {
				const next = await submit(form.data as z.infer<S>);
				if (next) form.data = { ...form.data, ...next };
			} catch (err) {
				if (!(err instanceof ApiError)) throw err;
				const fields = Object.entries(err.fields);
				for (const [field, message] of fields) {
					setError(form, field as '', message);
				}
				if (fields.length === 0) setMessage(form, err.message);
				form.valid = false;
			}
		}
	});
}
