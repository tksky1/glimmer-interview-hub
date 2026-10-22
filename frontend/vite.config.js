import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import { fileURLToPath } from 'node:url'

// https://vitejs.dev/config/
export default defineConfig({
  base: '/interview/',
  plugins: [
    vue(),
    // Element Plus 组件与 API 自动引入（与旧前端工程一致）
    // importStyle: false —— 样式统一由 main.js 引入的 element-plus/dist/index.css 提供，
    // 避免按需样式与全量样式重复打包
    AutoImport({
      resolvers: [ElementPlusResolver({ importStyle: false })]
    }),
    Components({
      resolvers: [ElementPlusResolver({ importStyle: false })]
    })
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
    extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue']
  },
  server: {
    host: '0.0.0.0',
    port: 4000,
    cors: true
  },
  build: {
    chunkSizeWarningLimit: 1500
  }
})
