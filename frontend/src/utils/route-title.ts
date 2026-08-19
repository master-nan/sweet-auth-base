import { primitiveText } from './primitive-text'

export const resolveRouteTitle = (title: unknown, translate: (key: string) => string) => {
  const value = primitiveText(title)
  return value.startsWith('router.') ? translate(value) : value
}
