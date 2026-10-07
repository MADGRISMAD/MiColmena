<script lang="ts">
	import LegalPage from '#lib/components/legal-page.svelte';
</script>

<LegalPage title="Términos de servicio" updated="[fecha]">
	<h2>1. Quiénes somos</h2>
	<p>
		MiColmena es un servicio de mesa de ayuda ofrecido por <strong>[Nombre o razón social]</strong>,
		con domicilio en <strong>[domicilio]</strong>. Al crear una cuenta aceptas estos términos.
	</p>

	<h2>2. La cuenta de tu empresa</h2>
	<ul>
		<li>Quien crea la cuenta declara tener autorización para hacerlo en nombre de la empresa.</li>
		<li>
			Eres responsable de las personas a las que das acceso y de mantener seguras sus contraseñas.
		</li>
		<li>
			Cada agente es una persona: una cuenta de agente no se comparte. Al entrar desde otro
			dispositivo se cierra la sesión anterior.
		</li>
		<li>El número de personas de la empresa que declaras debe ser veraz, pues define el precio.</li>
	</ul>

	<h2>3. Planes y pagos</h2>
	<ul>
		<li>
			El plan gratis incluye 1 agente para empresas de hasta 10 personas, sin vencimiento, mientras
			el servicio esté disponible.
		</li>
		<li>
			Los planes de pago se cobran en pesos mexicanos, más IVA, de forma [mensual o anual]
			anticipada. [Condiciones de facturación, renovación y cancelación].
		</li>
		<li>Podemos cambiar los precios avisándote con al menos [30] días de anticipación.</li>
	</ul>

	<h2>4. Tus datos y los de tus clientes</h2>
	<p>
		La información que tu empresa guarda en MiColmena (tickets, mensajes, archivos) es de tu
		empresa. La tratamos solo para prestarte el servicio, según el
		<a href="/privacidad" class="underline">aviso de privacidad</a>. Tu empresa es responsable de
		informar a sus propios clientes cómo usa sus datos.
	</p>

	<h2>5. Uso aceptable</h2>
	<p>
		No puedes usar MiColmena para actividades ilegales, enviar spam, intentar acceder a datos de
		otras empresas o afectar el funcionamiento del servicio. Podemos suspender cuentas que incumplan
		estos términos.
	</p>

	<h2>6. Disponibilidad y responsabilidad</h2>
	<p>
		Hacemos nuestro mejor esfuerzo para que el servicio esté disponible y respaldado, pero puede
		tener interrupciones. [Límites de responsabilidad].
	</p>

	<h2>7. Cancelación</h2>
	<p>
		Puedes dejar de usar el servicio cuando quieras. [Plazo durante el cual puedes descargar tu
		información y después del cual se elimina].
	</p>

	<h2>8. Ley aplicable</h2>
	<p>
		Estos términos se rigen por las leyes de los Estados Unidos Mexicanos. [Jurisdicción de los
		tribunales].
	</p>

	<h2>9. Contacto</h2>
	<p><strong>[email de contacto]</strong></p>
</LegalPage>
