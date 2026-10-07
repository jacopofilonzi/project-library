<script lang="ts">
  import { tick } from 'svelte'
  import { store, type Indexed } from '../lib/state.svelte'
  import { t, tn, ago } from '../lib/i18n/index.svelte'
  import { colorOf, initials } from '../lib/icons'
  import LauncherIcon from './LauncherIcon.svelte'

  let q = $state('')
  let global = $state(!store.path.length)
  let idx = $state(0)
  let input: HTMLInputElement | undefined = $state()
  let list: HTMLDivElement | undefined = $state()

  $effect(() => {
    input?.focus()
  })

  let here = $derived(store.path.at(-1) ?? store.rootLabel)
  let all = $derived([...store.index.values()])

  const esc = (s: string) => s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)
  function hl(text: string) {
    const terms = q.trim().split(/\s+/).filter(Boolean).map((x) => x.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
    const safe = esc(text)
    return terms.length ? safe.replace(new RegExp('(' + terms.map(esc).join('|') + ')', 'gi'), '<mark>$1</mark>') : safe
  }

  const inScope = (e: Indexed, scope: string[]) => scope.every((s, i) => e.parent[i] === s)
  function matches(e: Indexed) {
    const hay = (e.node.name + ' ' + (e.node.desc ?? '') + ' ' + e.parent.join('/')).toLowerCase()
    return q.toLowerCase().split(/\s+/).filter(Boolean).every((term) => hay.includes(term))
  }

  type Item = { kind: 'proj'; e: Indexed; launcher?: string; at?: number } | { kind: 'glob'; n: number }

  let view = $derived.by(() => {
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

  function go(i: number, launch: boolean) {
    const it = view.items[i]
    if (!it) return
    if (it.kind === 'glob') {
      global = true
      idx = 0
      return
    }
    store.paletteOpen = false
    store.goToPath(it.e.node.path)
    if (launch) store.open(it.e.node)
  }

  function onkey(e: KeyboardEvent) {
    const mod = e.ctrlKey || e.metaKey
    if (e.key === 'ArrowDown') { e.preventDefault(); idx = Math.min(view.items.length - 1, idx + 1) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); idx = Math.max(0, idx - 1) }
    else if (e.key === 'Enter') { e.preventDefault(); go(idx, mod) }
    else if (e.key === 'Tab') { e.preventDefault(); if (store.path.length) { global = !global; idx = 0 } }
    else if (e.key === 'Backspace' && !q && !global) { global = true; idx = 0 }
    else if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); store.paletteOpen = false }
  }

  let def = $derived(store.defaultLauncher)
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="pal-bg" onmousedown={(e) => e.target === e.currentTarget && (store.paletteOpen = false)}>
  <div class="pal" role="dialog" aria-modal="true" aria-label={t('palette.label')}>
    <div class="in">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="M20 20l-3.5-3.5"/></svg>
      {#if global}
        <span class="chip global">{t('palette.everywhere')}</span>
      {:else}
        <span class="chip" title={store.path.join('/')}>{t('palette.in', { name: here })}<button title={t('palette.toGlobal')} onclick={() => { global = true; input?.focus() }}>×</button></span>
      {/if}
      <input bind:this={input} bind:value={q} oninput={() => (idx = 0)} onkeydown={onkey} autocomplete="off" spellcheck="false" aria-label={t('palette.label')}
        placeholder={global ? t('palette.searchAll') : t('palette.searchIn', { name: here })} />
    </div>
    <div class="lst" role="listbox" bind:this={list}>
      <div class="sec">{view.section}</div>
      {#if view.recent && !view.items.length}
        <div class="none">{t('palette.noRecent')}</div>
      {:else if !view.total && !view.items.some((x) => x.kind === 'glob' && x.n)}
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
      <span>{t('palette.keys.move')}</span><span>{t('palette.keys.go')}</span>
      {#if def}<span>{t('palette.keys.open', { mod: store.mod, name: def.name })}</span>{/if}
      <span class="sp"></span>
      {#if store.path.length}<span>{t('palette.keys.scope')}</span>{/if}
      <span>{t('palette.keys.close')}</span>
    </div>
  </div>
</div>
