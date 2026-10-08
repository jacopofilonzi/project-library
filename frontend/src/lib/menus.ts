// Context menu items (right click, ⋯ button, + button of the columns).
import { store, type CtxItem } from './state.svelte'
import { t } from './i18n/index.svelte'
import type { Node } from './api'

function overrideItems(node: Node): CtxItem[] {
  const ov = store.cfg.overrides?.[node.path]
  if (ov) return [{ label: t('ctx.clearOverride'), run: () => store.setOverride(node.path, '') }]
  if (node.kind === 'project') return [{ label: t('ctx.markDir'), run: () => store.setOverride(node.path, 'dir') }]
  return [{ label: t('ctx.markProject'), run: () => store.setOverride(node.path, 'project') }]
}

export function nodeMenu(node: Node): CtxItem[] {
  const def = store.launcherFor(node).launcher
  const items: CtxItem[] = []
  if (def) items.push({ label: t('ctx.open', { name: def.name }), key: node.kind === 'project' ? '↵' : undefined, run: () => store.open(node) })
  items.push(
    { label: t('ctx.reveal'), key: store.mod + ' E', run: () => store.reveal(node.path) },
    { label: t('ctx.copyPath'), run: () => store.copyPath(node.path) },
  )
  if (node.kind === 'empty' && !node.missing) {
    items.push({ label: t('init.button') + '…', run: () => (store.dialog = { kind: 'init', path: node.path, name: node.name }) })
  }
  if (node.kind !== 'project' && !node.missing) {
    items.push(
      { label: t('ctx.newFolder'), run: () => (store.dialog = { kind: 'newFolder', parent: node.path }) },
      { label: t('ctx.cloneHere'), run: () => (store.dialog = { kind: 'clone', parent: node.path }) },
    )
  }
  items.push('-', ...overrideItems(node), '-')
  items.push(
    { label: t('ctx.rename'), key: 'F2', run: () => (store.dialog = { kind: 'rename', path: node.path, name: node.name }) },
    { label: t('ctx.delete'), key: store.os === 'darwin' ? '⌘⌫' : 'Del', danger: true, run: () => store.askDelete(node) },
  )
  return items
}

export function addMenu(dir: Node): CtxItem[] {
  return [
    { label: t('ctx.newFolder'), key: store.mod + ' Shift N', run: () => (store.dialog = { kind: 'newFolder', parent: dir.path }) },
    { label: t('ctx.cloneHere'), run: () => (store.dialog = { kind: 'clone', parent: dir.path }) },
  ]
}

export function openCtx(e: MouseEvent | { x: number; y: number }, items: CtxItem[]) {
  store.ctx = { x: e.x, y: e.y, items }
}
