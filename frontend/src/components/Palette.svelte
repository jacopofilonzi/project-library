<script lang="ts">
  import { tick, onMount, untrack } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import { store, type Indexed } from '../lib/state.svelte'
  import { t, tn, ago } from '../lib/i18n/index.svelte'
  import { colorOf, initials } from '../lib/icons'
  import { commands, runCommand, type Command } from '../lib/commands'
  import { lib } from '../lib/api'
  import LauncherIcon from './LauncherIcon.svelte'

  // spotlight: ricerca flottante, sempre su tutto Development; ↵ apre il progetto, Ctrl/⌘ ↵ lo mostra nell'app
  let { spotlight = false }: { spotlight?: boolean } = $props()

  let q = $state('')
  // valore iniziale: la ricerca flottante parte sempre da "ovunque", la palette da "ovunque" solo alla radice
  let global = $state(untrack(() => spotlight || !store.path.length))
  let idx = $state(0)
  let input: HTMLInputElement | undefined = $state()
  let list: HTMLDivElement | undefined = $state()

  $effect(() => {
    input?.focus()
  })

  // la ricerca flottante riparte da zero ogni volta che compare
  onMount(() => {
    if (!spotlight) return
    return Events.On('spotlight:open', () => {
      q = ''
      idx = 0
      tick().then(() => input?.focus())
    })
  })

  function close() {
    if (spotlight) lib.HideSpotlight()
    else store.paletteOpen = false
  }

  let here = $derived(store.path.at(-1) ?? store.rootLabel)
  let all = $derived([...store.index.values()])
  let commandMode = $derived(q.startsWith('>'))
  let cq = $derived(q.slice(1).trim().toLowerCase())

  const esc = (s: string) => s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)
  function hl(text: string, query = q) {
    const terms = query.trim().split(/\s+/).filter(Boolean).map((x) => x.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
    const safe = esc(text)
    return terms.length ? safe.replace(new RegExp('(' + terms.map(esc).join('|') + ')', 'gi'), '<mark>$1</mark>') : safe
  }

  const inScope = (e: Indexed, scope: string[]) => scope.every((s, i) => e.parent[i] === s)
  function matches(e: Indexed) {
    const hay = (e.node.name + ' ' + (e.node.desc ?? '') + ' ' + e.parent.join('/')).toLowerCase()
    return q.toLowerCase().split(/\s+/).filter(Boolean).every((term) => hay.includes(term))
  }

  type Item = { kind: 'proj'; e: Indexed; launcher?: string; at?: number } | { kind: 'glob'; n: number } | { kind: 'cmd'; c: Command }

  let view = $derived.by(() => {
    if (commandMode) {
      const items: Item[] = commands()
        .filter((c) => !cq || cq.split(/\s+/).every((term) => c.label.toLowerCase().includes(term)))
        .map((c) => ({ kind: 'cmd', c }))
      return { section: t('palette.commands'), items, recent: false, total: items.length }
    }
    // "ovunque" senza testo: ultimi 5 aperti con "Apri con"
    if (global && !q) {
      const items: Item[] = (store.cfg.recent ?? [])
        .map((r) => ({ r, e: store.index.get(r.path) }))
        .filter((x) => x.e)
        .slice(0, 5)
        .map((x) => ({ kind: 'proj', e: x.e!, launcher: x.r.launcher, at: x.r.at }))
      return { section: t('palette.recent'), items, recent: true, total: items.length }
    }
    const scope = global ? [] : store.path
    const found = all.filter((e) => inScope(e, scope) && matches(e)).sort((a, b) => a.node.name.localeCompare(b.node.name))
    const items: Item[] = (q ? found : found.slice(0, 8)).map((e) => ({ kind: 'proj', e }))
    if (!global && q) {
      const elsewhere = all.filter((e) => !inScope(e, scope) && matches(e)).length
      items.push({ kind: 'glob', n: elsewhere })
    }
    const section = q ? (global ? t('palette.everywhereSec') : t('palette.inPath', { path: store.path.join('/') || store.rootLabel })) + ' · ' + found.length : t('palette.inPath', { path: here })
    return { section, items, recent: false, total: found.length }
  })

  $effect(() => {
    view
    idx = Math.min(idx, Math.max(0, view.items.length - 1))
  })
  $effect(() => {
    idx
    tick().then(() => list?.querySelector('.it.on')?.scrollIntoView({ block: 'nearest' }))
  })

  // alt: Ctrl/⌘ ↵. Nella finestra principale apre con l'editor, nella ricerca flottante mostra nell'app.
  function go(i: number, alt: boolean) {
    const it = view.items[i]
    if (!it) return
    if (it.kind === 'glob') {
      global = true
      idx = 0
      return
    }
    if (it.kind === 'cmd') {
      close()
      runCommand(it.c)
      return
    }
    const node = it.e.node
    if (spotlight) {
      if (alt) lib.ShowInMain(node.path)
      else store.open(node)
      return
    }
    store.paletteOpen = false
    store.goToPath(node.path)
    if (alt) store.open(node)
  }

  function onkey(e: KeyboardEvent) {
    const mod = e.ctrlKey || e.metaKey
    if (e.key === 'ArrowDown') { e.preventDefault(); idx = Math.min(view.items.length - 1, idx + 1) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); idx = Math.max(0, idx - 1) }
    else if (e.key === 'Enter') { e.preventDefault(); go(idx, mod) }
    else if (e.key === 'Tab') { e.preventDefault(); if (!spotlight && store.path.length && !commandMode) { global = !global; idx = 0 } }
    else if (e.key === 'Backspace' && !q && !global) { global = true; idx = 0 }
    else if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); close() }
  }

  // launcher del progetto evidenziato (regole e scelte per progetto comprese)
  let def = $derived.by(() => {
    const it = view.items[idx]
    return it?.kind === 'proj' ? store.launcherFor(it.e.node).launcher : store.defaultLauncher
  })
</script>

{#snippet panel()}
  <div class="pal" class:floating={spotlight} role="dialog" aria-modal="true" aria-label={t('palette.label')}>
    <div class="in">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="M20 20l-3.5-3.5"/></svg>
      {#if commandMode}
        <span class="chip global">{t('palette.commandChip')}</span>
      {:else if !spotlight}
        {#if global}
          <span class="chip global">{t('palette.everywhere')}</span>
        {:else}
          <span class="chip" title={store.path.join('/')}>{t('palette.in', { name: here })}<button title={t('palette.toGlobal')} onclick={() => { global = true; input?.focus() }}>×</button></span>
        {/if}
      {/if}
      <input bind:this={input} bind:value={q} oninput={() => (idx = 0)} onkeydown={onkey} autocomplete="off" spellcheck="false" aria-label={t('palette.label')}
        placeholder={spotlight ? t('palette.spotlightPlaceholder') : global ? t('palette.searchAll') : t('palette.searchIn', { name: here })} />
    </div>
    <div class="lst" role="listbox" bind:this={list}>
      <div class="sec">{view.section}</div>
      {#if view.recent && !view.items.length}
        <div class="none">{t('palette.noRecent')}</div>
      {:else if commandMode && !view.items.length}
        <div class="none">{t('palette.noCommands')}</div>
      {:else if !commandMode && !view.total && !view.items.some((x) => x.kind === 'glob' && x.n)}
        <div class="none">{global ? t('palette.none') : t('palette.noneIn', { name: here })}{q ? t('palette.noneFor', { q }) : ''}.</div>
      {/if}
      {#each view.items as it, i}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div class="it" class:on={i === idx} class:glob={it.kind === 'glob'} role="option" aria-selected={i === idx} tabindex="-1"
          onmousemove={() => (idx = i)} onclick={(e) => go(i, e.ctrlKey || e.metaKey)}>
          {#if it.kind === 'glob'}
            <div class="ib glob">⌕</div>
            <div class="m"><div class="t">{t('palette.global', { q })}</div><div class="s">{tn('palette.globalSub', it.n, { name: here })}</div></div>
            <span class="w">⇥</span>
          {:else if it.kind === 'cmd'}
            <div class="ib glob">›</div>
            <div class="m"><div class="t">{@html hl(it.c.label, cq)}</div></div>
            {#if it.c.key}<span class="w">{it.c.key}</span>{/if}
          {:else}
            {@const p = it.e.node}
            {@const where = (global ? it.e.parent : it.e.parent.slice(store.path.length)).join('/')}
            {@const l = it.launcher ? store.cfg.launchers.find((x) => x.id === it.launcher) : undefined}
            <div class="ib" style="background:{colorOf(p.lang)}">{initials(p.name)}</div>
            <div class="m">
              <div class="t">{@html hl(p.name)}</div>
              <div class="s">{#if view.recent}{it.e.parent.join('/') || store.rootLabel}{:else}{@html p.desc ? hl(p.desc) : esc(p.lang || '')}{/if}</div>
            </div>
            {#if view.recent}
              <span class="w">{#if l}<LauncherIcon launcher={l} />{/if}{it.at ? ago(it.at) : ''}</span>
            {:else}
              <span class="w">{where}</span>
            {/if}
          {/if}
        </div>
      {/each}
    </div>
    <div class="ft">
      <span>{t('palette.keys.move')}</span>
      {#if commandMode}
        <span>{t('palette.keys.run')}</span>
      {:else if spotlight}
        {#if def}<span>{t('palette.keys.openWith', { name: def.name })}</span>{/if}
        <span>{t('palette.keys.showInApp', { mod: store.mod })}</span>
      {:else}
        <span>{t('palette.keys.go')}</span>
        {#if def}<span>{t('palette.keys.open', { mod: store.mod, name: def.name })}</span>{/if}
      {/if}
      <span class="sp"></span>
      {#if !commandMode}<span>{t('palette.keys.commands')}</span>{/if}
      {#if !spotlight && store.path.length && !commandMode}<span>{t('palette.keys.scope')}</span>{/if}
      <span>{t('palette.keys.close')}</span>
    </div>
  </div>
{/snippet}

{#if spotlight}
  {@render panel()}
{:else}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="pal-bg" onmousedown={(e) => e.target === e.currentTarget && close()}>
    {@render panel()}
  </div>
{/if}
