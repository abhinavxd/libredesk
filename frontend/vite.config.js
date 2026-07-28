import { fileURLToPath } from 'url'
import { createRequire } from 'module'
import path from 'path'
import autoprefixer from 'autoprefixer'
import tailwind from 'tailwindcss'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const require = createRequire(import.meta.url)

export default defineConfig(({ mode, command }) => {
  const isWidget = mode === 'widget'
  const appPath = isWidget ? 'apps/widget' : 'apps/main'

  const apiTarget = process.env.LD_API_TARGET || 'http://127.0.0.1:9000'
  const wsTarget = process.env.LD_WS_TARGET || 'ws://127.0.0.1:9000'
  const mainPort = process.env.LD_DEV_PORT ? Number(process.env.LD_DEV_PORT) : 8000

  // Load shared tailwind config but scope content to current app only,
  // so each app's CSS bundle doesn't include unused classes from the other.
  const tailwindConfig = require('./tailwind.config.cjs')
  const scopedContent = [
    `./apps/${isWidget ? 'widget' : 'main'}/src/**/*.{js,ts,vue}`,
    './shared-ui/**/*.{js,ts,vue}',
  ]

  return {
    base: isWidget && command === 'build' ? '/widget/' : '/',
    css: {
      preprocessorOptions: {
        scss: {
          api: 'modern',
        },
      },
      postcss: {
        plugins: [tailwind({ ...tailwindConfig, content: scopedContent }), autoprefixer()],
      },
    },
    root: path.resolve(__dirname, appPath),
    publicDir: path.resolve(__dirname, 'public'),
    // Separate cache per app to avoid stale/conflicting caches.
    cacheDir: path.resolve(__dirname, `node_modules/.vite-${isWidget ? 'widget' : 'main'}`),
    server: {
      cors: { origin: "*" },
      // Allow access to parent dir so shared-ui imports work in dev.
      fs: {
        allow: [path.resolve(__dirname)],
      },
      port: isWidget ? 8001 : mainPort,
      proxy: {
        '/api': {
          target: apiTarget,
          changeOrigin: true,
        },
        '/widget.js': {
          target: apiTarget,
          changeOrigin: true,
        },
        '/static': {
          target: apiTarget,
          changeOrigin: true,
        },
        '/logout': {
          target: apiTarget,
          changeOrigin: true,
        },
        '/uploads': {
          target: apiTarget,
          changeOrigin: true,
        },
        '/ws': {
          target: wsTarget,
          ws: true,
          changeOrigin: true,
        },
        '/widget/ws': {
          target: wsTarget,
          ws: true,
          changeOrigin: true,
        }
      },
    },
    build: {
      outDir: isWidget
        ? path.resolve(__dirname, 'dist/widget')
        : path.resolve(__dirname, 'dist/main'),
      emptyOutDir: true,
      chunkSizeWarningLimit: 600,
      rollupOptions: {
        output: {
          manualChunks: {
            'vue-vendor': ['vue', 'vue-router', 'pinia'],
            'radix': ['radix-vue', 'reka-ui'],
            'icons': ['lucide-vue-next', '@radix-icons/vue'],
            'utils': ['@vueuse/core', 'clsx', 'tailwind-merge', 'class-variance-authority'],
            'forms': ['vee-validate', '@vee-validate/zod', 'zod'],
            'misc': ['axios', 'date-fns', 'mitt', 'qs', 'vue-i18n'],
            // Main-app-only chunks - widget doesn't use these libraries.
            ...(!isWidget && {
              'charts': ['@unovis/ts', '@unovis/vue'],
              'editor': [
                '@tiptap/vue-3',
                '@tiptap/starter-kit',
                '@tiptap/extension-image',
                '@tiptap/extension-link',
                '@tiptap/extension-placeholder',
                '@tiptap/extension-table',
                '@tiptap/extension-table-cell',
                '@tiptap/extension-table-header',
                '@tiptap/extension-table-row',
              ],
              'codemirror': ['codemirror', '@codemirror/lang-html', '@codemirror/lang-javascript', '@codemirror/theme-one-dark'],
              'table': ['@tanstack/vue-table'],
            }),
          },
        },
      },
    },
    plugins: [vue()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, `${appPath}/src`),
        '@main': path.resolve(__dirname, 'apps/main/src'),
        '@widget': path.resolve(__dirname, 'apps/widget/src'),
        '@shared-ui': path.resolve(__dirname, 'shared-ui'),
      },
    },
  }
})
