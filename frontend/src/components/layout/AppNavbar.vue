<template>
  <div class="bg-white text-gray-900 p-4 border-b border-gray-200 flex items-center justify-between shadow-lg">
    <!-- Left section with menu button and title -->
    <div class="flex items-center space-x-4">
      <button class="p-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 focus:outline-none">
        <img src="https://placehold.co/24x24?text=Menu" alt="Menu" class="w-6 h-6">
      </button>
      <h1 class="text-lg font-semibold">My Dashboard</h1>
    </div>

    <!-- Right section with action buttons and search input -->
    <div class="flex items-center space-x-4">
      <button class="p-2 bg-yellow-600 text-white rounded-md hover:bg-yellow-700 focus:outline-none">Get started</button>
      <div class="relative">
        <input type="text" placeholder="Search" class="p-2 border border-gray-300 rounded-full bg-gray-50 text-gray-900 focus:outline-none focus:border-yellow-500">
        
      </div>
      <div class="flex space-x-2">
        <button class="p-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 focus:outline-none">
          <img src="https://placehold.co/24x24?text=Icon8" alt="Icon8" class="w-6 h-6 rounded-full">
        </button>
        <button class="p-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 focus:outline-none">
          <img src="https://placehold.co/24x24?text=Icon9" alt="Icon9" class="w-6 h-6 rounded-full">
        </button>
        <button class="p-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 focus:outline-none">
          <img src="https://placehold.co/24x24?text=Icon10" alt="Icon10" class="w-6 h-6 rounded-full">
        </button>
        <button class="p-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 focus:outline-none">
          <img src="https://placehold.co/24x24?text=Icon11" alt="Icon11" class="w-6 h-6 rounded-full">
        </button>
        <button class="p-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 focus:outline-none">
          <img src="https://placehold.co/24x24?text=Icon12" alt="Icon12" class="w-6 h-6 rounded-full">
        </button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'AppNavbar'
};
</script>

<style scoped>
/* No se necesitan estilos adicionales ya que todo se maneja con Tailwind CSS */
</style>
