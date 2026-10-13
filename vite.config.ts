import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    watch: {
      // Polling avoids Linux inotify exhaustion in large workspaces.
      usePolling: true,
      interval: 1000,
    },
  },
})
