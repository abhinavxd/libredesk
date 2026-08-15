// moveWithin moves the item one step in the list, returning false when it is already at the edge.
export function moveWithin(list, id, direction) {
  const from = list.findIndex((item) => item.id === id)
  const to = from + direction
  if (from === -1 || to < 0 || to >= list.length) return false
  list.splice(to, 0, list.splice(from, 1)[0])
  return true
}
