// Esquemas de validación de los formularios. Repiten las reglas del backend para avisar
// al usuario antes de enviar; el backend sigue validando por su cuenta.
// Se usa zod/mini, la variante de Zod pensada para el navegador: misma validación, mucho menos peso.
import * as z from 'zod/mini';
import { TICKET_PRIORITIES, TICKET_STATUSES } from '#lib/api/types.js';

export const loginSchema = z.object({
	email: z.email('Introduce un email válido'),
	password: z.string().check(z.minLength(1, 'Introduce tu contraseña'))
});

export const registerSchema = z
	.object({
		name: z
			.string()
			.check(
				z.trim(),
				z.minLength(1, 'Introduce tu nombre'),
				z.maxLength(100, 'Máximo 100 caracteres')
			),
		email: z.email('Introduce un email válido'),
		password: z
			.string()
			.check(
				z.minLength(8, 'Debe tener al menos 8 caracteres'),
				z.maxLength(72, 'Debe tener como máximo 72 caracteres')
			),
		confirm: z.string()
	})
	.check(
		z.refine((data) => data.password === data.confirm, {
			message: 'Las contraseñas no coinciden',
			path: ['confirm']
		})
	);

export const ticketSchema = z.object({
	title: z
		.string()
		.check(
			z.trim(),
			z.minLength(1, 'El título es obligatorio'),
			z.maxLength(200, 'Máximo 200 caracteres')
		),
	description: z.string().check(z.maxLength(20000, 'La descripción es demasiado larga')),
	priority: z.enum(TICKET_PRIORITIES),
	category: z.string().check(z.trim(), z.maxLength(100, 'Máximo 100 caracteres'))
});

export const commentSchema = z.object({
	body: z
		.string()
		.check(
			z.trim(),
			z.minLength(1, 'Escribe un comentario'),
			z.maxLength(20000, 'Es demasiado largo')
		),
	internal: z.boolean()
});

export const ticketStatusSchema = z.enum(TICKET_STATUSES);

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;
export type TicketInput = z.infer<typeof ticketSchema>;
export type CommentInput = z.infer<typeof commentSchema>;

/** Valores iniciales de cada formulario. */
export const emptyLogin: LoginInput = { email: '', password: '' };
export const emptyRegister: RegisterInput = { name: '', email: '', password: '', confirm: '' };
export const emptyTicket: TicketInput = {
	title: '',
	description: '',
	priority: 'medium',
	category: ''
};
export const emptyComment: CommentInput = { body: '', internal: false };
