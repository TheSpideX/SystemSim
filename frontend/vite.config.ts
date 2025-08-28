import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0', // Expose on all network interfaces
    port: 5173,      // Default port
    allowedHosts: [
      'localhost',
      '127.0.0.1',
      'saty.local',    // mDNS hostname
      '.local',        // Allow all .local domains
    ],
  },
})
