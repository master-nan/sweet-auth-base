type SchemeDeletedListener = (id: number) => void

const deletedListeners = new Set<SchemeDeletedListener>()

export const notifyQuerySchemeDeleted = (id: number) => {
  deletedListeners.forEach((listener) => listener(id))
}

export const subscribeQuerySchemeDeleted = (listener: SchemeDeletedListener) => {
  deletedListeners.add(listener)
  return () => deletedListeners.delete(listener)
}
