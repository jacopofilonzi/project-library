<script lang="ts">
  import { onMount } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import { store } from './lib/state.svelte'
  import { t, tn } from './lib/i18n/index.svelte'
  import Header from './components/Header.svelte'
  import Columns from './components/Columns.svelte'
  import ProjectCard from './components/ProjectCard.svelte'
  import EmptyCard from './components/EmptyCard.svelte'
  import Palette from './components/Palette.svelte'
  import Settings from './components/Settings.svelte'
  import Wizard from './components/Wizard.svelte'
  import Dialogs from './components/dialogs/Dialogs.svelte'
  import ContextMenu from './components/ContextMenu.svelte'
  import Toasts from './components/Toasts.svelte'
  import type { Node } from './lib/api'

  onMount(() => {
    store.init()
    return Events.On('shortcut:settings', () => {
      if (store.wizardOpen) return
      if (store.settingsOpen) store.settingsOpen = false
      else store.openSettings()
    })
  })

  let overlay = $derived(store.paletteOpen || store.settingsOpen || store.wizardOpen || !!store.dialog || !!store.ctx)

  // ---------- navigazione da tastiera nelle colonne ----------
  function kids(names: string[]): Node[] {
    return (store.nodeAt(names)?.children ?? []).filter((c): c is Node => !!c)
  }
  function moveSelection(d: number) {
    // l'elemento attivo è il progetto selezionato o l'ultima cartella del percorso
    const inLast = store.sel !== null || !store.path.length
    const parent = inLast ? store.path : store.path.slice(0, -1)
    const list = kids(parent)
    if (!list.length) return
    const current = store.sel ?? store.path.at(-1)
    const i = list.findIndex((c) => c.name === current)
    const next = list[Math.max(0, Math.min(list.length - 1, i < 0 ? 0 : i + d))]
    if (next.kind === 'project') store.go(parent, next.name)
    else store.go([...parent, next.name])
  }
  function moveIn() {
    if (store.sel) return
    const first = kids(store.path)[0]
    if (!first) return
    if (first.kind === 'project') store.go(store.path, first.name)
    else store.go([...store.path, first.name])
  }
  function moveOut() {
    if (store.sel) store.go(store.path)
    else if (store.path.length) store.go(store.path.slice(0, -1))
  }

  function onkey(e: KeyboardEvent) {
    const mod = e.ctrlKey || e.metaKey
    const k = e.key.toLowerCase()
    if (store.wizardOpen) return
    if (mod && k === 'k') {
      e.preventDefault()
      store.settingsOpen = false
      store.paletteOpen = !store.paletteOpen
      return
    }
    // Ctrl/⌘ P e Ctrl/⌘ , arrivano dal backend come evento "shortcut:settings" (vedi main.go);
    // qui si blocca solo l'eventuale stampa.
    if (mod && (k === 'p' || k === ',')) {
      e.preventDefault()
      return
    }
    if (e.key === 'Escape') {
      if (store.ctx) store.ctx = null
      else if (store.dialog && store.dialog.kind !== 'clone') store.dialog = null
      else if (store.paletteOpen) store.paletteOpen = false
      else if (store.settingsOpen) store.settingsOpen = false
      return
    }
    if (overlay) return
    const target = e.target as HTMLElement
    if (target.matches('input, textarea, select')) return

    const tgt = store.target
    const node = store.selected ?? store.current
    if (mod && e.shiftKey && k === 'n') {
      e.preventDefault()
      const dir = store.current
      if (dir && dir.kind !== 'root' && !dir.missing) store.dialog = { kind: 'newFolder', parent: dir.path }
    } else if (e.key === 'Delete' || (e.metaKey && e.key === 'Backspace')) {
      if (node && store.path.length) { e.preventDefault(); store.askDelete(node) }
    } else if (e.key === 'F2') {
      if (node && store.path.length) { e.preventDefault(); store.dialog = { kind: 'rename', path: node.path, name: node.name } }
    } else if (e.key === 'Enter' && !target.matches('button')) {
      if (tgt) { e.preventDefault(); store.open(tgt) }
    } else if (mod && /^[1-9]$/.test(e.key)) {
      const l = store.enabledLaunchers[+e.key - 1]
      if (l && tgt) { e.preventDefault(); store.open(tgt, l.id) }
    } else if (mod && k === 'e') {
      e.preventDefault()
      const p = tgt?.path ?? store.current?.path
      if (p) store.reveal(p)
    } else if (e.key === '/') {
      e.preventDefault()
      store.paletteOpen = true
    } else if (e.key === 'ArrowDown') { e.preventDefault(); moveSelection(1) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); moveSelection(-1) }
    else if (e.key === 'ArrowRight') { e.preventDefault(); moveIn() }
    else if (e.key === 'ArrowLeft') { e.preventDefault(); moveOut() }
  }
</script>

<svelte:window onkeydown={onkey} />

{#if store.ready}
  <div class="app">
    <Header />
    <main class="view">
      <Columns />
      {#if store.selected}
        {#key store.selected.path}<ProjectCard node={store.selected} />{/key}
      {:else if store.current && store.path.length && store.current.kind === 'empty' && !store.current.missing}
        <EmptyCard node={store.current} />
      {:else}
        <div class="empty">
          {#if store.current?.count}
            {tn('summary.projects', store.current.count, { name: store.path.at(-1) ?? store.rootLabel })}
          {:else}
            {t('summary.none')}
          {/if}
        </div>
      {/if}
    </main>
  </div>

  {#if store.paletteOpen}<Palette />{/if}
  {#if store.settingsOpen}<Settings />{/if}
  <Dialogs />
  <ContextMenu />
  {#if store.wizardOpen}<Wizard />{/if}
{/if}
<Toasts />
