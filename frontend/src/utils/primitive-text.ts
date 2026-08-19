export const primitiveText = (value: unknown, fallback = '') => {
  switch (typeof value) {
    case 'string':
      return value
    case 'number':
    case 'boolean':
    case 'bigint':
      return String(value)
    default:
      return fallback
  }
}
