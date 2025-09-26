// vite.config.js
import { fileURLToPath, URL } from "node:url";
import autoprefixer from "file:///home/crx/LIBREDESK/libredesk/frontend/node_modules/.pnpm/autoprefixer@10.4.21_postcss@8.5.6/node_modules/autoprefixer/lib/autoprefixer.js";
import tailwind from "file:///home/crx/LIBREDESK/libredesk/frontend/node_modules/.pnpm/tailwindcss@3.4.17/node_modules/tailwindcss/lib/index.js";
import { defineConfig } from "file:///home/crx/LIBREDESK/libredesk/frontend/node_modules/.pnpm/vite@5.4.20_@types+node@22.10.5_sass@1.83.1_stylus@0.57.0/node_modules/vite/dist/node/index.js";
import vue from "file:///home/crx/LIBREDESK/libredesk/frontend/node_modules/.pnpm/@vitejs+plugin-vue@5.2.1_vite@5.4.20_@types+node@22.10.5_sass@1.83.1_stylus@0.57.0__vue@3.5.13_typescript@5.7.3_/node_modules/@vitejs/plugin-vue/dist/index.mjs";
var __vite_injected_original_import_meta_url = "file:///home/crx/LIBREDESK/libredesk/frontend/vite.config.js";
var vite_config_default = defineConfig({
  css: {
    postcss: {
      plugins: [tailwind(), autoprefixer()]
    }
  },
  server: {
    port: 8e3,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:9000"
      },
      "/logout": {
        target: "http://127.0.0.1:9000"
      },
      "/uploads": {
        target: "http://127.0.0.1:9000"
      },
      "/ws": {
        target: "ws://127.0.0.1:9000",
        ws: true
      }
    }
  },
  build: {
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        manualChunks: {
          "vue-vendor": ["vue", "vue-router", "pinia"],
          "radix": ["radix-vue", "reka-ui"],
          "icons": ["lucide-vue-next", "@radix-icons/vue"],
          "utils": ["@vueuse/core", "clsx", "tailwind-merge", "class-variance-authority"],
          "charts": ["@unovis/ts", "@unovis/vue"],
          "editor": ["@tiptap/vue-3", "@tiptap/starter-kit", "@tiptap/extension-image", "@tiptap/extension-link", "@tiptap/extension-placeholder", "@tiptap/extension-table", "@tiptap/extension-table-cell", "@tiptap/extension-table-header", "@tiptap/extension-table-row"],
          "forms": ["vee-validate", "@vee-validate/zod", "zod"],
          "table": ["@tanstack/vue-table"],
          "misc": ["axios", "date-fns", "mitt", "qs", "vue-i18n"]
        }
      }
    }
  },
  plugins: [
    vue()
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", __vite_injected_original_import_meta_url))
    }
  }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcuanMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvaG9tZS9jcngvTElCUkVERVNLL2xpYnJlZGVzay9mcm9udGVuZFwiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9maWxlbmFtZSA9IFwiL2hvbWUvY3J4L0xJQlJFREVTSy9saWJyZWRlc2svZnJvbnRlbmQvdml0ZS5jb25maWcuanNcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfaW1wb3J0X21ldGFfdXJsID0gXCJmaWxlOi8vL2hvbWUvY3J4L0xJQlJFREVTSy9saWJyZWRlc2svZnJvbnRlbmQvdml0ZS5jb25maWcuanNcIjtpbXBvcnQgeyBmaWxlVVJMVG9QYXRoLCBVUkwgfSBmcm9tICdub2RlOnVybCdcbmltcG9ydCBhdXRvcHJlZml4ZXIgZnJvbSAnYXV0b3ByZWZpeGVyJ1xuaW1wb3J0IHRhaWx3aW5kIGZyb20gJ3RhaWx3aW5kY3NzJ1xuaW1wb3J0IHsgZGVmaW5lQ29uZmlnIH0gZnJvbSAndml0ZSdcbmltcG9ydCB2dWUgZnJvbSAnQHZpdGVqcy9wbHVnaW4tdnVlJ1xuXG5leHBvcnQgZGVmYXVsdCBkZWZpbmVDb25maWcoe1xuICBjc3M6IHtcbiAgICBwb3N0Y3NzOiB7XG4gICAgICBwbHVnaW5zOiBbdGFpbHdpbmQoKSwgYXV0b3ByZWZpeGVyKCldLFxuICAgIH0sXG4gIH0sXG4gIHNlcnZlcjoge1xuICAgIHBvcnQ6IDgwMDAsXG4gICAgcHJveHk6IHtcbiAgICAgICcvYXBpJzoge1xuICAgICAgICB0YXJnZXQ6ICdodHRwOi8vMTI3LjAuMC4xOjkwMDAnLFxuICAgICAgfSxcbiAgICAgICcvbG9nb3V0Jzoge1xuICAgICAgICB0YXJnZXQ6ICdodHRwOi8vMTI3LjAuMC4xOjkwMDAnLFxuICAgICAgfSxcbiAgICAgICcvdXBsb2Fkcyc6IHtcbiAgICAgICAgdGFyZ2V0OiAnaHR0cDovLzEyNy4wLjAuMTo5MDAwJyxcbiAgICAgIH0sXG4gICAgICAnL3dzJzoge1xuICAgICAgICB0YXJnZXQ6ICd3czovLzEyNy4wLjAuMTo5MDAwJyxcbiAgICAgICAgd3M6IHRydWUsXG4gICAgICB9LFxuICAgIH0sXG4gIH0sXG4gIGJ1aWxkOiB7XG4gICAgY2h1bmtTaXplV2FybmluZ0xpbWl0OiA2MDAsXG4gICAgcm9sbHVwT3B0aW9uczoge1xuICAgICAgb3V0cHV0OiB7XG4gICAgICAgIG1hbnVhbENodW5rczoge1xuICAgICAgICAgICd2dWUtdmVuZG9yJzogWyd2dWUnLCAndnVlLXJvdXRlcicsICdwaW5pYSddLFxuICAgICAgICAgICdyYWRpeCc6IFsncmFkaXgtdnVlJywgJ3Jla2EtdWknXSxcbiAgICAgICAgICAnaWNvbnMnOiBbJ2x1Y2lkZS12dWUtbmV4dCcsICdAcmFkaXgtaWNvbnMvdnVlJ10sXG4gICAgICAgICAgJ3V0aWxzJzogWydAdnVldXNlL2NvcmUnLCAnY2xzeCcsICd0YWlsd2luZC1tZXJnZScsICdjbGFzcy12YXJpYW5jZS1hdXRob3JpdHknXSxcbiAgICAgICAgICAnY2hhcnRzJzogWydAdW5vdmlzL3RzJywgJ0B1bm92aXMvdnVlJ10sXG4gICAgICAgICAgJ2VkaXRvcic6IFsnQHRpcHRhcC92dWUtMycsICdAdGlwdGFwL3N0YXJ0ZXIta2l0JywgJ0B0aXB0YXAvZXh0ZW5zaW9uLWltYWdlJywgJ0B0aXB0YXAvZXh0ZW5zaW9uLWxpbmsnLCAnQHRpcHRhcC9leHRlbnNpb24tcGxhY2Vob2xkZXInLCAnQHRpcHRhcC9leHRlbnNpb24tdGFibGUnLCAnQHRpcHRhcC9leHRlbnNpb24tdGFibGUtY2VsbCcsICdAdGlwdGFwL2V4dGVuc2lvbi10YWJsZS1oZWFkZXInLCAnQHRpcHRhcC9leHRlbnNpb24tdGFibGUtcm93J10sXG4gICAgICAgICAgJ2Zvcm1zJzogWyd2ZWUtdmFsaWRhdGUnLCAnQHZlZS12YWxpZGF0ZS96b2QnLCAnem9kJ10sXG4gICAgICAgICAgJ3RhYmxlJzogWydAdGFuc3RhY2svdnVlLXRhYmxlJ10sXG4gICAgICAgICAgJ21pc2MnOiBbJ2F4aW9zJywgJ2RhdGUtZm5zJywgJ21pdHQnLCAncXMnLCAndnVlLWkxOG4nXVxuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuICB9LFxuICBwbHVnaW5zOiBbXG4gICAgdnVlKCksXG4gIF0sXG4gIHJlc29sdmU6IHtcbiAgICBhbGlhczoge1xuICAgICAgJ0AnOiBmaWxlVVJMVG9QYXRoKG5ldyBVUkwoJy4vc3JjJywgaW1wb3J0Lm1ldGEudXJsKSlcbiAgICB9XG4gIH0sXG59KSJdLAogICJtYXBwaW5ncyI6ICI7QUFBb1MsU0FBUyxlQUFlLFdBQVc7QUFDdlUsT0FBTyxrQkFBa0I7QUFDekIsT0FBTyxjQUFjO0FBQ3JCLFNBQVMsb0JBQW9CO0FBQzdCLE9BQU8sU0FBUztBQUpvSyxJQUFNLDJDQUEyQztBQU1yTyxJQUFPLHNCQUFRLGFBQWE7QUFBQSxFQUMxQixLQUFLO0FBQUEsSUFDSCxTQUFTO0FBQUEsTUFDUCxTQUFTLENBQUMsU0FBUyxHQUFHLGFBQWEsQ0FBQztBQUFBLElBQ3RDO0FBQUEsRUFDRjtBQUFBLEVBQ0EsUUFBUTtBQUFBLElBQ04sTUFBTTtBQUFBLElBQ04sT0FBTztBQUFBLE1BQ0wsUUFBUTtBQUFBLFFBQ04sUUFBUTtBQUFBLE1BQ1Y7QUFBQSxNQUNBLFdBQVc7QUFBQSxRQUNULFFBQVE7QUFBQSxNQUNWO0FBQUEsTUFDQSxZQUFZO0FBQUEsUUFDVixRQUFRO0FBQUEsTUFDVjtBQUFBLE1BQ0EsT0FBTztBQUFBLFFBQ0wsUUFBUTtBQUFBLFFBQ1IsSUFBSTtBQUFBLE1BQ047QUFBQSxJQUNGO0FBQUEsRUFDRjtBQUFBLEVBQ0EsT0FBTztBQUFBLElBQ0wsdUJBQXVCO0FBQUEsSUFDdkIsZUFBZTtBQUFBLE1BQ2IsUUFBUTtBQUFBLFFBQ04sY0FBYztBQUFBLFVBQ1osY0FBYyxDQUFDLE9BQU8sY0FBYyxPQUFPO0FBQUEsVUFDM0MsU0FBUyxDQUFDLGFBQWEsU0FBUztBQUFBLFVBQ2hDLFNBQVMsQ0FBQyxtQkFBbUIsa0JBQWtCO0FBQUEsVUFDL0MsU0FBUyxDQUFDLGdCQUFnQixRQUFRLGtCQUFrQiwwQkFBMEI7QUFBQSxVQUM5RSxVQUFVLENBQUMsY0FBYyxhQUFhO0FBQUEsVUFDdEMsVUFBVSxDQUFDLGlCQUFpQix1QkFBdUIsMkJBQTJCLDBCQUEwQixpQ0FBaUMsMkJBQTJCLGdDQUFnQyxrQ0FBa0MsNkJBQTZCO0FBQUEsVUFDblEsU0FBUyxDQUFDLGdCQUFnQixxQkFBcUIsS0FBSztBQUFBLFVBQ3BELFNBQVMsQ0FBQyxxQkFBcUI7QUFBQSxVQUMvQixRQUFRLENBQUMsU0FBUyxZQUFZLFFBQVEsTUFBTSxVQUFVO0FBQUEsUUFDeEQ7QUFBQSxNQUNGO0FBQUEsSUFDRjtBQUFBLEVBQ0Y7QUFBQSxFQUNBLFNBQVM7QUFBQSxJQUNQLElBQUk7QUFBQSxFQUNOO0FBQUEsRUFDQSxTQUFTO0FBQUEsSUFDUCxPQUFPO0FBQUEsTUFDTCxLQUFLLGNBQWMsSUFBSSxJQUFJLFNBQVMsd0NBQWUsQ0FBQztBQUFBLElBQ3REO0FBQUEsRUFDRjtBQUNGLENBQUM7IiwKICAibmFtZXMiOiBbXQp9Cg==
