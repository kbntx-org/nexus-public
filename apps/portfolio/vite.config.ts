import react from '@vitejs/plugin-react';
import autoprefixer from 'autoprefixer';
import path from 'node:path';
import tailwindcss from 'tailwindcss';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react()],
  css: {
    postcss: {
      plugins: [tailwindcss(), autoprefixer()]
    }
  },
  resolve: {
    alias: { '@': path.resolve(import.meta.dirname, './src') }
  },
  build: {
    outDir: path.resolve(import.meta.dirname, '../../dist/apps/portfolio'),
    emptyOutDir: true
  },
  server: {
    host: true,
    port: 4200,
    allowedHosts: true
  }
});
