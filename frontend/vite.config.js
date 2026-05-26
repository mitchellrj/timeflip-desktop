import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return undefined;
          }
          if (id.includes('/react/') || id.includes('/react-dom/')) {
            return 'react-vendor';
          }
          if (id.includes('/lucide-react/') || id.includes('/lucide/')) {
            return 'icons';
          }
          if (id.includes('/@wailsio/')) {
            return 'wails';
          }
          return 'vendor';
        },
      },
    },
  },
});
