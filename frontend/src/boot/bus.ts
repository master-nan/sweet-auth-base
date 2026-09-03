import { defineBoot } from '#q-app';
import { EventBus } from 'quasar'

export default defineBoot(({ app }) => {
  const bus = new EventBus()
  app.provide('bus', bus)
})
