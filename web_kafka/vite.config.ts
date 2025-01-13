import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import dotenv from 'dotenv';

// Load environment variables using dotenv
dotenv.config();

// Si necesitas cargar las variables para el modo de desarrollo o producción
const env = loadEnv('development', process.cwd(), ['VITE_']);

console.log('Env:', env);
console.log('API URL:', env.VITE_API_URL); // Imprime las variables de entorno en la consola del servidor
console.log("******************************");
console.log('Current working directory:', process.cwd()); // Imprime el directorio de trabajo en la consola del servidor
console.log("******************************");

export default defineConfig({
  plugins: [react()],
 
  server: {
    host: '0.0.0.0',
    port: 5173,
  },
  build: {
    target: 'esnext', // Especifica que el objetivo es la última versión de ECMAScript (compatible con ESM)
    minify: 'esbuild', // Usa minificación compatible con ESM
    lib: {
      entry: './src/index.tsx', // Asegúrate de que la entrada sea el archivo correcto (index.tsx o app.tsx)
      formats: ['es'], // Define el formato de salida como ESM
    },
  },
});