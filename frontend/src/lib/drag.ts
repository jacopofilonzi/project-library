// Drag & drop to move folders and projects: rows are dragged, folders (rows, column headers,
// breadcrumb) accept the drop. Dropping never moves by itself: it opens the move dialog to confirm.
//
// Built on pointer events, not on HTML5 drag & drop: with the custom title bar the WebView is
// composition-hosted, and there HTML5 drag & drop goes through Windows (OLE) and is refused.
import { store, uiZoom } from './state.svelte'

/** a press becomes a drag after this many pixels */
const THRESHOLD = 5
/** hovering a folder row while dragging opens it, so you can go deeper */
const OPEN_AFTER = 800

// folder path → navigation into it (registered by the rows that open on hover)
const openers = new Map<string, () => void>()

let pending: { path: string; label: string; x: number; y: number } | null = null
let ghost: HTMLDivElement | null = null
let openTimer: ReturnType<typeof setTimeout> | undefined

/** drag source: a folder or project row */
export function dragSource(path: string, label: string) {
  return {
    onpointerdown: (e: PointerEvent) => {
      if (e.button !== 0 || store.dragging) return
      pending = { path, label, x: e.clientX, y: e.clientY }
      window.addEventListener('pointermove', onMove)
      window.addEventListener('pointerup', onUp)
      window.addEventListener('keydown', onKey, true)
    },
  }
}

/** drop target: the folder at dest. open (optional) navigates into it after a short hover. */
export function dropTarget(dest: string, open?: () => void) {
  if (open) openers.set(dest, open)
  return { 'data-drop': dest }
}

function targetAt(x: number, y: number): string {
  const el = document.elementFromPoint(x, y)?.closest<HTMLElement>('[data-drop]')
  const dest = el?.dataset.drop ?? ''
  return dest && store.canMoveTo(store.dragging, dest) ? dest : ''
}

function onMove(e: PointerEvent) {
  if (!pending) return
  if (!store.dragging) {
    if (Math.hypot(e.clientX - pending.x, e.clientY - pending.y) < THRESHOLD) return
    store.dragging = pending.path
    document.body.classList.add('pl-dragging')
    ghost = document.createElement('div')
    ghost.className = 'drag-ghost'
    ghost.textContent = pending.label
    document.body.append(ghost)
  }
  // the ghost is placed in CSS pixels, the pointer is in screen pixels (see the interface scale)
  const z = uiZoom()
  ghost!.style.left = `${e.clientX / z + 12}px`
  ghost!.style.top = `${e.clientY / z + 12}px`
  const dest = targetAt(e.clientX, e.clientY)
  if (dest !== store.dropTarget) {
    store.dropTarget = dest
    clearTimeout(openTimer)
    const open = dest ? openers.get(dest) : undefined
    if (open) openTimer = setTimeout(open, OPEN_AFTER)
  }
}

function onUp(e: PointerEvent) {
  const wasDragging = !!store.dragging
  const src = store.dragging
  const dest = wasDragging ? targetAt(e.clientX, e.clientY) : ''
  end()
  if (!wasDragging) return
  // the release ends on a button: its click must not run after a drag
  window.addEventListener('click', swallow, { capture: true, once: true })
  setTimeout(() => window.removeEventListener('click', swallow, true), 0)
  const node = store.nodeByPath(src)
  if (node && dest) store.dialog = { kind: 'move', node, dest }
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && store.dragging) {
    e.stopPropagation()
    end()
  }
}

function swallow(e: Event) {
  e.stopPropagation()
  e.preventDefault()
}

function end() {
  pending = null
  clearTimeout(openTimer)
  store.dragging = ''
  store.dropTarget = ''
  document.body.classList.remove('pl-dragging')
  ghost?.remove()
  ghost = null
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', onUp)
  window.removeEventListener('keydown', onKey, true)
}
