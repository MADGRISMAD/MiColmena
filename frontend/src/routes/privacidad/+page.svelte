<script lang="ts">
	import LegalPage from '#lib/components/legal-page.svelte';
</script>

<LegalPage title="Aviso de privacidad" updated="[fecha]">
	<h2>1. Quién es responsable de tus datos</h2>
	<p>
		<strong>[Nombre o razón social]</strong>, con domicilio en <strong>[domicilio completo]</strong>
		(«MiColmena»), es responsable del tratamiento de los datos personales que recabamos a través de este
		sitio, conforme a la Ley Federal de Protección de Datos Personales en Posesión de los Particulares.
	</p>
	<p>
		Cuando una empresa usa MiColmena para atender a sus propios clientes, esa empresa es la
		responsable de los datos de sus clientes y MiColmena los trata por su cuenta, solo para
		prestarle el servicio. Si eres cliente de una de esas empresas, consulta también su aviso de
		privacidad.
	</p>

	<h2>2. Qué datos tratamos</h2>
	<ul>
		<li>Identificación y contacto: nombre, email, teléfono o WhatsApp y empresa.</li>
		<li>Datos de la cuenta: contraseña (guardada cifrada), rol y preferencias de avisos.</li>
		<li>
			Datos de uso: tickets, mensajes, archivos adjuntos y registros técnicos (dirección IP, fecha y
			hora) necesarios para la seguridad del servicio.
		</li>
		<li>
			Datos de facturación, cuando contratas un plan de pago: [RFC, razón social, régimen fiscal].
		</li>
	</ul>
	<p>No solicitamos datos personales sensibles.</p>

	<h2>3. Para qué los usamos</h2>
	<p>Finalidades necesarias para el servicio:</p>
	<ul>
		<li>Crear y administrar tu cuenta y la de tu empresa.</li>
		<li>Prestar el servicio de mesa de ayuda y enviarte los avisos que configures.</li>
		<li>Atender solicitudes de demo, cotizaciones y soporte.</li>
		<li>Facturar y cobrar los planes de pago.</li>
		<li>Proteger la seguridad de las cuentas y prevenir abusos.</li>
	</ul>
	<p>
		Finalidades adicionales: [por ejemplo, enviarte novedades del producto]. Puedes negarte a estas
		escribiendo a <strong>[email de privacidad]</strong>; negarte no afecta el servicio.
	</p>

	<h2>4. Con quién los compartimos</h2>
	<p>
		Con proveedores que nos ayudan a prestar el servicio y que solo pueden usarlos para eso:
		[hospedaje de servidores, envío de correo, procesador de pagos, facturación electrónica]. No
		vendemos tus datos. Solo los entregaremos a autoridades cuando la ley lo exija.
	</p>

	<h2>5. Tus derechos ARCO</h2>
	<p>
		Puedes acceder a tus datos, rectificarlos, cancelarlos u oponerte a su uso, así como revocar tu
		consentimiento, escribiendo a <strong>[email de privacidad]</strong> con tu nombre, el derecho que
		quieres ejercer y una copia de tu identificación. Te responderemos en un plazo máximo de [20] días
		hábiles.
	</p>

	<h2>6. Cookies y almacenamiento local</h2>
	<p>
		Usamos el almacenamiento local de tu navegador para mantener tu sesión iniciada y recordar el
		tema claro u oscuro. No usamos cookies de publicidad ni de rastreo de terceros.
	</p>

	<h2>7. Cambios a este aviso</h2>
	<p>
		Publicaremos cualquier cambio en esta página con su fecha de actualización y, si es importante,
		te avisaremos por email.
	</p>
</LegalPage>
