import { transformAsync } from '@babel/core'
import linguiMacro from '@lingui/babel-plugin-lingui-macro'
import { makeConfig } from '@lingui/conf'
import config from './lingui.config.mjs'

// Vue first compiles the SFC; Lingui then replaces descriptors with stable IDs.
export function lingui() {
  return {
    name: 'libredesk-lingui',
    enforce: 'post',
    async transform(code, id) {
      if (id.includes('/node_modules/') || !code.includes('@lingui/core/macro')) return
      const result = await transformAsync(code, {
        filename: id.split('?')[0] + '.js',
        babelrc: false,
        configFile: false,
        sourceMaps: true,
        plugins: [[linguiMacro, { linguiConfig: makeConfig(config) }]],
      })
      return { code: result.code, map: result.map }
    },
  }
}
