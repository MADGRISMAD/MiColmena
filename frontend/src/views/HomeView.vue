<template>
  <div class="flex h-screen">
    <AppSidebar />
    
    <div class="flex-1 flex flex-col">
      <AppNavbar />
      
      <div class="flex-1 p-6 overflow-y-auto">
        <div class="container mx-auto">
          <h1 class="text-4xl font-bold mb-6 text-gray-900">Bienvenido a MiColmena</h1>
          <p class="text-xl text-gray-600 mb-8">
            MiColmena es una plataforma integral para gestionar tickets de soporte y atención al cliente.
            Organiza y gestiona tus tickets de manera eficiente y mejora la experiencia de tus usuarios.
          </p>
          
          <div class="flex justify-center space-x-4 mb-8">
            <router-link to="/register" class="bg-yellow-500 text-white px-6 py-3 rounded-lg shadow-md hover:bg-yellow-600 transition-colors duration-300">
              Registrarse
            </router-link>
            <router-link to="/login" class="bg-black text-white px-6 py-3 rounded-lg shadow-md hover:bg-gray-900 transition-colors duration-300">
              Iniciar sesión
            </router-link>
          </div>
          
          <h2 class="text-3xl font-semibold mb-6 text-gray-900">Pon la IA a tu servicio con MiColmena</h2>
          <p class="text-lg text-gray-600 mb-8">
            Descubre cómo nuestra plataforma de IA combina tecnología tradicional y nueva para que tus equipos trabajen en lo que realmente importa.
          </p>
          <div class="flex justify-center space-x-4 mb-8">
            <a href="#" class="text-blue-600 underline">Ver vídeo (1:00)</a>
            <a href="#" class="text-blue-600 underline">Explorar demostraciones</a>
          </div>
          
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 mb-8">
            <!-- Empleados -->
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">Haz que la IA trabaje al servicio de los empleados</h3>
              <p class="text-lg mb-4">Ofrece experiencias escalables a los empleados para aumentar el compromiso, fomentar el desarrollo de talentos e impulsar la productividad.</p>
              <ul class="list-disc list-inside">
                <li>HR Service Delivery</li>
                <li>Workplace Service Delivery</li>
                <li>Soluciones para empleados</li>
              </ul>
            </div>
            <!-- Clientes -->
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">Haz que la IA trabaje al servicio de los clientes</h3>
              <p class="text-lg mb-4">Con la IA y la automatización, puedes facultar a los agentes y técnicos de campo para que presten servicio de manera proactiva, lo que ayuda a reducir los esfuerzos de los clientes.</p>
              <ul class="list-disc list-inside">
                <li>Customer Service Management</li>
                <li>Field Service Management</li>
                <li>Soluciones para clientes</li>
              </ul>
            </div>
            <!-- Operaciones de TI -->
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">Haz que la IA trabaje al servicio de las operaciones de TI</h3>
              <p class="text-lg mb-4">Une procesos en una sola plataforma para ofrecer servicios digitales resilientes, experiencias extraordinarias y una mayor productividad.</p>
              <ul class="list-disc list-inside">
                <li>IT Operations Management</li>
                <li>IT Service Management</li>
                <li>Soluciones de operaciones de TI</li>
              </ul>
            </div>
            <!-- Desarrollo de aplicaciones -->
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">Haz que la IA trabaje al servicio del desarrollo de aplicaciones</h3>
              <p class="text-lg mb-4">Emplea la IA y la automatización para modernizar los procesos manuales y optimizar los flujos de trabajo a lo largo de toda tu organización.</p>
              <ul class="list-disc list-inside">
                <li>App Engine</li>
                <li>Automation Engine</li>
                <li>Soluciones de automatización</li>
              </ul>
            </div>
            <!-- Ciberseguridad y riesgo -->
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">Haz que la IA trabaje al servicio de la ciberseguridad y riesgo</h3>
              <p class="text-lg mb-4">Protege el crecimiento, la resiliencia y la productividad de negocios cuando se presentan amenazas, regulaciones y cambios.</p>
              <ul class="list-disc list-inside">
                <li>Security Operations</li>
                <li>Governance, Risk, and Compliance</li>
                <li>Soluciones de ciberseguridad</li>
              </ul>
            </div>
            <!-- Cadena de suministro y finanzas -->
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">Haz que la IA trabaje al servicio de la cadena de suministro y finanzas</h3>
              <p class="text-lg mb-4">Reimagina lo que es posible cuando puedes conectar fácilmente personas, procesos, datos y tecnología.</p>
              <ul class="list-disc list-inside">
                <li>Accounts Payable Operations</li>
                <li>Supplier Lifecycle Management</li>
                <li>Soluciones de finanzas</li>
              </ul>
            </div>
          </div>
          
          <h2 class="text-3xl font-semibold mb-6 text-gray-900">Haz que la IA trabaje al servicio de las personas</h2>
          <p class="text-lg text-gray-600 mb-8">
            MiColmena lleva la IA a todos los rincones de tu negocio, lo que permite que tu personal sea más productivo, más receptivo y más innovador.
          </p>
          <div class="flex justify-center space-x-4 mb-8">
            <a href="#" class="text-blue-600 underline">Descubrir cómo</a>
            <a href="#" class="text-blue-600 underline">Reproducir vídeo</a>
          </div>
          
          <h2 class="text-3xl font-semibold mb-6 text-gray-900">Ve cómo más de 8100 clientes globales ayudan a que el mundo funcione</h2>
          <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-8">
            <img src="https://placehold.co/100x40?text=BT" alt="BT logo" class="object-contain h-10">
            <img src="https://placehold.co/100x40?text=Kraft+Heinz" alt="Kraft Heinz logo" class="object-contain h-10">
            <img src="https://placehold.co/100x40?text=Uber" alt="Uber logo" class="object-contain h-10">
            <img src="https://placehold.co/100x40?text=NTT+Data" alt="NTT Data logo" class="object-contain h-10">
            <img src="https://placehold.co/100x40?text=Lowes" alt="Lowes logo" class="object-contain h-10">
            <img src="https://placehold.co/100x40?text=Dropbox" alt="Dropbox logo" class="object-contain h-10">
            <img src="https://placehold.co/100x40?text=Bimbo" alt="Bimbo logo" class="object-contain h-10">
          </div>
          
          <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md mb-8">
            <h3 class="text-2xl font-semibold mb-4">La eficiencia digital lleva el pan al mundo</h3>
            <p class="text-lg mb-4">Con información basada en datos de más de 200 instalaciones de fabricación de pan, Grupo Bimbo aumenta el control de la calidad a escala mundial.</p>
            <a href="#" class="text-blue-600 underline">Leer historia</a>
          </div>
          
          <div class="flex justify-between items-center mb-8">
            <h2 class="text-3xl font-semibold text-gray-900">Novedades en MiColmena</h2>
            <a href="#" class="text-blue-600 underline">Ver todas las noticias</a>
          </div>
          <div class="space-y-4">
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">Ganancias del primer trimestre de 2024</h3>
              <p class="text-lg mb-4">MiColmena anuncia sus resultados financieros para el primer trimestre de 2024.</p>
              <a href="#" class="text-blue-600 underline">Leer comunicado de prensa</a>
            </div>
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">MiColmena amplía su liderazgo en GenAI con su último lanzamiento</h3>
              <p class="text-lg mb-4">Las mejoras de Now Assist y MiColmena Impact AI Accelerators mejoran la productividad y aumentan el valor de las inversiones en IA.</p>
              <a href="#" class="text-blue-600 underline">Leer el comunicado de prensa</a>
            </div>
            <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md">
              <h3 class="text-2xl font-semibold mb-4">Última versión de plataforma aplica IA generativa</h3>
              <p class="text-lg mb-4">Conectamos soluciones de IA generativa a flujos de trabajo generando resultados comerciales reales.</p>
              <a href="#" class="text-blue-600 underline">Leer comunicado</a>
            </div>
          </div>
          
          <div class="flex justify-between items-center mt-8">
            <h2 class="text-3xl font-semibold text-gray-900">Informe de analista</h2>
            <a href="#" class="text-blue-600 underline">Obtener el informe</a>
          </div>
          <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md mt-4">
            <h3 class="text-2xl font-semibold mb-4">MiColmena, líder en Forrester Wave™ para GRC</h3>
            <p class="text-lg mb-4">Descubre por qué se nombró a MiColmena líder en The Forrester Wave™: Governance, Risk, and Compliance Platforms, Q4, 2023.</p>
            <a href="#" class="text-blue-600 underline">Obtener el informe</a>
          </div>
          
          <div class="flex justify-between items-center mt-8">
            <h2 class="text-3xl font-semibold text-gray-900">Knowledge 2024</h2>
            <a href="#" class="text-blue-600 underline">Explorar las sesiones</a>
          </div>
          <div class="bg-gray-50 text-gray-900 p-6 rounded-lg shadow-md mt-4">
            <h3 class="text-2xl font-semibold mb-4">La inteligencia ocupa la agenda</h3>
            <p class="text-lg mb-4">Laboratorios de IA. Demostraciones inmersivas. Luminarias invitadas enfocadas en el futuro. Únete a nosotros en Knowledge 2024 para liberar tu potencial.</p>
            <a href="#" class="text-blue-600 underline">Explorar las sesiones</a>
          </div>
          
          <div class="flex justify-center mt-8">
            <a href="#" class="text-blue-600 underline">Obtén las últimas actualizaciones de MiColmena</a>
          </div>
          <div class="flex justify-center mt-4">
            <input type="email" placeholder="Correo electrónico" class="p-2 border border-gray-300 rounded-lg shadow-sm focus:outline-none focus:border-blue-500">
            <button class="bg-blue-600 text-white px-6 py-2 rounded-lg shadow-md ml-2 hover:bg-blue-700 transition-colors duration-300">Suscribirse</button>
          </div>
          
          <footer class="mt-8">
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
              <div>
                <h4 class="text-lg font-semibold mb-4 text-gray-900">Empresa</h4>
                <ul class="text-gray-600">
                  <li><a href="#" class="hover:underline">Acerca de</a></li>
                  <li><a href="#" class="hover:underline">Oportunidades laborales</a></li>
                  <li><a href="#" class="hover:underline">Ubicaciones</a></li>
                  <li><a href="#" class="hover:underline">Socios</a></li>
                  <li><a href="#" class="hover:underline">Proveedores</a></li>
                  <li><a href="#" class="hover:underline">Inversionistas</a></li>
                  <li><a href="#" class="hover:underline">Código de ética</a></li>
                  <li><a href="#" class="hover:underline">Sala de prensa</a></li>
                </ul>
              </div>
              <div>
                <h4 class="text-lg font-semibold mb-4 text-gray-900">Recursos</h4>
                <ul class="text-gray-600">
                  <li><a href="#" class="hover:underline">Casos de clientes</a></li>
                  <li><a href="#" class="hover:underline">Investigación de MiColmena</a></li>
                  <li><a href="#" class="hover:underline">Formación y certificación</a></li>
                  <li><a href="#" class="hover:underline">Comunidad de usuarios</a></li>
                  <li><a href="#" class="hover:underline">Portal del desarrollador</a></li>
                  <li><a href="#" class="hover:underline">Documentación de producto</a></li>
                  <li><a href="#" class="hover:underline">Accesibilidad de productos</a></li>
                </ul>
              </div>
              <div>
                <h4 class="text-lg font-semibold mb-4 text-gray-900">Servicios</h4>
                <ul class="text-gray-600">
                  <li><a href="#" class="hover:underline">Servicios</a></li>
                  <li><a href="#" class="hover:underline">Portal Now Support</a></li>
                </ul>
              </div>
              <div>
                <h4 class="text-lg font-semibold mb-4 text-gray-900">Contacto</h4>
                <ul class="text-gray-600">
                  <li><a href="#" class="hover:underline">Solicita información o programa una demostración</a></li>
                  <li><a href="#" class="hover:underline">Comunicarte con nosotros</a></li>
                </ul>
              </div>
            </div>
            <div class="mt-8 text-gray-600 text-center">
              <p>&copy; 2024 MiColmena. Todos los derechos reservados.</p>
              <ul class="flex justify-center space-x-4 mt-4">
                <li><a href="#" class="hover:underline">Términos y condiciones</a></li>
                <li><a href="#" class="hover:underline">RGPD</a></li>
                <li><a href="#" class="hover:underline">Declaración de privacidad</a></li>
                <li><a href="#" class="hover:underline">Política de cookies</a></li>
                <li><a href="#" class="hover:underline">Preferencias de cookies</a></li>
                <li><a href="#" class="hover:underline">Mapa de sitio</a></li>
                <li><a href="#" class="hover:underline">Continuidad empresarial</a></li>
                <li><a href="#" class="hover:underline">Accesibilidad</a></li>
              </ul>
            </div>
          </footer>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import AppNavbar from '@/components/layout/AppNavbar.vue';
import AppSidebar from '@/components/layout/AppSidebar.vue';

export default {
  name: 'HomeView',
  components: {
    AppNavbar,
    AppSidebar
  }
};
</script>

<style scoped>
/* Estilos específicos de Home si es necesario */
</style>
