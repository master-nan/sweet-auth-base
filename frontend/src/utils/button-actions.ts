import type { MenuButton } from 'src/api/services/sys-menu'

export type PageActionHandler<Row> = (row: Row | undefined, button: MenuButton) => void
export type PageActionHandlers<Row> = Partial<Record<string, PageActionHandler<Row>>>

export const dispatchPageAction = <Row>(
  button: MenuButton,
  handlers: PageActionHandlers<Row>,
  row?: Row,
) => {
  const handler = handlers[button.event_action]
  if (!handler) return false
  handler(row, button)
  return true
}
