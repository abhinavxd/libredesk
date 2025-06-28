import { getCurrentInstance } from 'vue'

export function useEmitter() {
  const instance = getCurrentInstance()
  return instance.appContext.config.globalProperties.emitter
}
