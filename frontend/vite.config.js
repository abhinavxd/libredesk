import { fileURLToPath } from 'url'
import { createRequire } from 'module'
import path from 'path'
import autoprefixer from 'autoprefixer'
import tailwind from 'tailwindcss'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const require = createRequire(import.meta.url)

export default defineConfig(({ mode }) => {
  const isWidget = mode === 'widget'
  const appPath = isWidget ? 'apps/widget' : 'apps/main'

  // Load shared tailwind config but scope content to current app only,
  // so each app's CSS bundle doesn't include unused classes from the other.
  const tailwindConfig = require('./tailwind.config.cjs')
  const scopedContent = [
    `./apps/${isWidget ? 'widget' : 'main'}/src/**/*.{js,ts,vue}`,
    './shared-ui/**/*.{js,ts,vue}',
  ]

  return {
    base: isWidget ? '/widget/' : '/',
    css: {
      postcss: {
        plugins: [tailwind({ ...tailwindConfig, content: scopedContent }), autoprefixer()],
      },
    },
    root: path.resolve(__dirname, appPath),
    // Separate cache per app to avoid stale/conflicting caches.
    cacheDir: path.resolve(__dirname, `node_modules/.vite-${isWidget ? 'widget' : 'main'}`),
    server: {
      // Allow access to parent dir so shared-ui imports work in dev.
      fs: {
        allow: [path.resolve(__dirname)],
      },
      port: isWidget ? 8001 : 8000,
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:9000',
        },
        '/widget.js': {
          target: 'http://127.0.0.1:9000',
        },
        '/logout': {
          target: 'http://127.0.0.1:9000',
        },
        '/uploads': {
          target: 'http://127.0.0.1:9000',
        },
        '/ws': {
          target: 'ws://127.0.0.1:9000',
          ws: true,
        },
        '/widget/ws': {
          target: 'ws://127.0.0.1:9000',
          ws: true,
        }
      },
    },
    build: {
      outDir: isWidget
        ? path.resolve(__dirname, 'dist/widget')
        : path.resolve(__dirname, 'dist/main'),
      emptyOutDir: true,
      rollupOptions: {
        output: {
          manualChunks: {
            'vue-vendor': ['vue', 'vue-router', 'pinia'],
            'radix': ['radix-vue', 'reka-ui'],
            'icons': ['lucide-vue-next', '@radix-icons/vue'],
            'utils': ['@vueuse/core', 'clsx', 'tailwind-merge', 'class-variance-authority'],
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
            'forms': ['vee-validate', '@vee-validate/zod', 'zod'],
            'table': ['@tanstack/vue-table'],
            'misc': ['axios', 'date-fns', 'mitt', 'qs', 'vue-i18n'],
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
