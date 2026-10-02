export const resolvePreChatForm = (config = {}, isVisitor = true) => {
  const audience = isVisitor ? config.visitors : config.users
  if (!audience || typeof audience !== 'object') return config

  return {
    ...config,
    enabled: Boolean(config.enabled && (audience.enabled ?? true)),
    title: audience.title ?? config.title,
    fields: audience.fields ?? config.fields
  }
}
