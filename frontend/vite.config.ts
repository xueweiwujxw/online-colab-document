import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  optimizeDeps: {
    exclude: ['@casualoffice/sheets/xlsx'],
  },
  server: {
    host: '0.0.0.0',
    port: 3000,
    // WebSocket Origin validation on the backend intentionally only accepts
    // FRONTEND_ORIGIN. Do not silently fall back to another Vite port.
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
      '/office-collab': {
        target: 'ws://localhost:1234',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/office-collab/, ''),
        rewriteWsOrigin: true,
        ws: true,
      },
    },
  },
});
