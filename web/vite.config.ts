import { defineConfig } from 'vite';
import path from 'path';
import tailwindcss from '@tailwindcss/vite';
import { tanstackRouter } from '@tanstack/router-plugin/vite';
import react from '@vitejs/plugin-react';
import { nodePolyfills } from 'vite-plugin-node-polyfills';

export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://backend:8080',
        changeOrigin: true,
      },
    },
  },
  assetsInclude: ['**/*.md'],
  plugins: [
    {
      name: 'reload',
      configureServer(server) {
        const { ws, watcher } = server;
        watcher.on('change', (file) => {
          if (file.endsWith('.md')) {
            ws.send({
              type: 'full-reload',
            });
          }
        });
      },
    },
    tanstackRouter({
      target: 'react',
      autoCodeSplitting: true,
    }),
    react(),
    tailwindcss(),
    nodePolyfills({
      globals: {
        Buffer: true,
      },
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
});
