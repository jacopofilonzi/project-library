// Drag & drop to move folders and projects: rows are dragged, folders (rows, column headers,
// breadcrumb) accept the drop. Dropping never moves by itself: it opens the move dialog to confirm.
import { store } from './state.svelte'

const TYPE = 'application/x-project-library-path'

/** drag source: a folder or project row */
export function dragSource(path: string) {
  return {
    draggable: true,
    ondragstart: (e: DragEvent) => {
      e.dataTransfer?.setData(TYPE, path)
      if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
      store.dragging = path
    },
    ondragend: () => {
      store.dragging = ''
      store.dropTarget = ''
    },
  }
}

// hovering a folder row while dragging opens it, so you can go deeper
let openTimer: ReturnType<typeof setTimeout> | undefined

/** drop target: the folder at dest. open (optional) navigates into it after a short hover. */
export function dropTarget(dest: string, open?: () => void) {
  const ok = () => store.canMoveTo(store.dragging, dest)
  return {
    ondragover: (e: DragEvent) => {
      if (!store.dragging || !ok()) return
      e.preventDefault()
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
      if (store.dropTarget !== dest) {
        store.dropTarget = dest
        clearTimeout(openTimer)
        if (open) openTimer = setTimeout(open, 800)
      }
    },
    ondragleave: () => {
      if (store.dropTarget === dest) store.dropTarget = ''
      clearTimeout(openTimer)
    },
    ondrop: (e: DragEvent) => {
      e.preventDefault()
      clearTimeout(openTimer)
      const src = e.dataTransfer?.getData(TYPE) || store.dragging
      store.dragging = ''
      store.dropTarget = ''
      const node = store.nodeByPath(src)
      if (node && store.canMoveTo(src, dest)) store.dialog = { kind: 'move', node, dest }
    },
  }
}
