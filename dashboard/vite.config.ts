import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/wessley-mvp/',
  build: { outDir: '../docs', emptyOutDir: false },
  resolve: { alias: { "@": path.resolve(__dirname, "./src") } },
})
