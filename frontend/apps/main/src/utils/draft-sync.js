// Keep unsaved drafts in memory and retry the latest version. Writes for a key
// are serialized, including deletes, so an older save cannot resurrect a draft.
export function createDraftSync({ write, onState, retryDelay = 5000 }) {
  const pending = new Map()

  async function drain(key, entry) {
    if (entry.running) return
    clearTimeout(entry.timer)
    entry.running = true
    onState(key, 'saving')
    try {
      while (pending.get(key) === entry) {
        const revision = entry.revision
        await write(entry.uuid, entry.type, entry.draft)
        if (revision !== entry.revision) continue
        pending.delete(key)
        onState(key, 'saved')
        break
      }
    } catch {
      if (pending.get(key) !== entry) return
      onState(key, 'error')
      entry.timer = setTimeout(() => drain(key, entry), retryDelay)
    } finally {
      entry.running = false
    }
  }

  return {
    save(uuid, type, draft) {
      const key = `${uuid}::${type}`
      const entry = pending.get(key) || { uuid, type, revision: 0 }
      entry.draft = draft
      entry.revision++
      pending.set(key, entry)
      drain(key, entry)
    },
    retry(uuid, type) {
      const entries = uuid ? [[`${uuid}::${type}`, pending.get(`${uuid}::${type}`)]] : pending
      for (const [key, entry] of entries) if (entry) drain(key, entry)
    },
    dispose() {
      for (const entry of pending.values()) clearTimeout(entry.timer)
      pending.clear()
    }
  }
}
