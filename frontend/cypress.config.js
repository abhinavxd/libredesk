/* eslint-env node */
import { defineConfig } from 'cypress'
import { start, control } from './cypress/support/metaMock.mjs'
import { startSetupServer, stopSetupServer } from './cypress/support/setupServer.mjs'

const metaMockPort = Number(process.env.META_MOCK_PORT) || 9099

export default defineConfig({
  e2e: {
    specPattern: 'cypress/e2e/**/*.{cy,spec}.{js,jsx,ts,tsx}',
    baseUrl: 'http://localhost:9000',
    async setupNodeEvents(on, config) {
      // The app reaches this stand-in Graph API when whatsapp.api_url points at this port.
      await start(metaMockPort)
      on('task', {
        'metaMock:reset': control.reset,
        'metaMock:requests': control.requests,
        'metaMock:failSend': control.failSend,
        'metaMock:failValidate': control.failValidate,
        'metaMock:failTemplateEdit': control.failTemplateEdit,
        'metaMock:failTemplateDelete': control.failTemplateDelete,
        'metaMock:putTemplate': control.putTemplate,
        'metaMock:setTemplateStatus': control.setTemplateStatus,
        'metaMock:putMedia': control.putMedia,
        'metaMock:sign': control.sign
      })
      if (config.env.FRESH_INSTALL) {
        const { origin, code } = await startSetupServer()
        config.baseUrl = origin
        config.env.SETUP_CODE = code
        on('after:run', stopSetupServer)
      }
      return config
    }
  },
  component: {
    specPattern: 'apps/main/src/**/__tests__/*.{cy,spec}.{js,ts,jsx,tsx}',
    devServer: {
      framework: 'vue',
      bundler: 'vite'
    }
  }
})
