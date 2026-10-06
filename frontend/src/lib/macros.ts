/** Reemplaza las variables de una respuesta guardada. */
export function fillMacro(
	body: string,
	vars: { solicitante: string; agente: string; ticket: number }
): string {
	const firstName = (name: string) => name.trim().split(/\s+/)[0] ?? name;
	return body
		.replaceAll('{{solicitante}}', firstName(vars.solicitante))
		.replaceAll('{{agente}}', firstName(vars.agente))
		.replaceAll('{{ticket}}', `#${vars.ticket}`);
}

/**
 * Si el texto antes del cursor termina en "@algo", devuelve ese "algo" (para sugerir menciones).
 * Devuelve null si no se está escribiendo una mención.
 */
export function mentionQuery(textBeforeCaret: string): string | null {
	const match = /(?:^|\s)@([\p{L}\p{N}_-]*)$/u.exec(textBeforeCaret);
	return match ? match[1] : null;
}
