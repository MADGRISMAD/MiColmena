import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import path from 'path'; // Asegúrate de importar path

export default defineConfig({
  plugins: [vue()],
  server: {
    // En desarrollo, las llamadas a /api van al backend en Go.
    proxy: {
      '/api': 'http://localhost:8080'
    }
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src') // Define correctamente la ruta de alias
    }
  }
});
