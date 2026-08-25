/* eslint-env node */
import { defineConfig } from 'cypress'
import { start, control } from './cypress/support/metaMock.mjs'

const metaMockPort = Number(process.env.META_MOCK_PORT) || 9099

export default defineConfig({
  e2e: {
    specPattern: 'cypress/e2e/**/*.{cy,spec}.{js,jsx,ts,tsx}',
    baseUrl: 'http://localhost:9000',
    async setupNodeEvents(on) {
      // The app reaches this stand-in Graph API when whatsapp.api_url points at this port.
      await start(metaMockPort).catch((err) => {
        if (err.code !== 'EADDRINUSE') throw err
      })
      on('task', {
        'metaMock:reset': control.reset,
        'metaMock:requests': control.requests,
        'metaMock:failSend': control.failSend,
        'metaMock:failValidate': control.failValidate,
        'metaMock:putMedia': control.putMedia,
        'metaMock:sign': control.sign
      })
    }
  },
  component: {
    specPattern: 'src/**/__tests__/*.{cy,spec}.{js,ts,jsx,tsx}',
    devServer: {
      framework: 'vue',
      bundler: 'vite'
    }
  }
})
