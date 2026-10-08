<script lang="ts">
  import { untrack } from 'svelte'
  import { store } from '../../lib/state.svelte'
  import { t } from '../../lib/i18n/index.svelte'
  import { icons } from '../../lib/icons'
  import { lib, errMessage, type Node } from '../../lib/api'

  // moves node into another folder. dest is preset when the dialog opens from a drag & drop;
  // it can still be changed from the list, and nothing moves until the confirmation.
  let { node, dest: initial }: { node: Node; dest?: string } = $props()
  const n = untrack(() => node)

  let dest = $state(untrack(() => initial ?? ''))
  let q = $state('')
  let busy = $state(false)
  let search: HTMLInputElement | undefined = $state()
  let confirmBtn: HTMLButtonElement | undefined = $state()
  // from a drag the destination is known: focus on the confirmation, otherwise on the search
  $effect(() => (untrack(() => initial) ? confirmBtn : search)?.focus())

  type Target = { path: string; label: string }
  // every folder of the tree (roots included) with its label; targets are the ones where node can go
  const folders: Target[] = []
  const walk = (x: Node, names: string[]) => {
    if (x.kind === 'project' || x.missing) return
    if (x.kind !== 'root') folders.push({ path: x.path, label: names.join(' / ') })
    for (const c of x.children ?? []) if (c) walk(c, x.kind === 'root' ? [c.name] : [...names, c.name])
  }
  if (store.tree) walk(store.tree, store.tree.kind === 'root' ? [] : [store.tree.name])
  const targets = folders.filter((x) => store.canMoveTo(n.path, x.path))
  const labelOf = (p: string) => folders.find((x) => x.path === p)?.label ?? p
  const fromLabel = labelOf(n.path.slice(0, Math.max(n.path.lastIndexOf('\\'), n.path.lastIndexOf('/'))))

  let filtered = $derived.by(() => {
    const words = q.toLowerCase().split(/\s+/).filter(Boolean)
    return targets.filter((x) => words.every((w) => x.label.toLowerCase().includes(w)))
  })

  async function run() {
    if (!dest || busy) return
    busy = true
    try {
      const to = await lib.Move(n.path, dest)
      store.dialog = null
      await store.afterMove(n.path, to)
      store.toast(t('move.done', { name: n.name, dest: labelOf(dest) }))
    } catch (e) {
      busy = false
      store.toast(errMessage(e), true)
    }
  }
</script>

<h3>{t('move.title', { name: n.name })}</h3>
<div class="bd">
  <input type="text" bind:this={search} bind:value={q} placeholder={t('move.search')} autocomplete="off" spellcheck="false" disabled={busy}
    onkeydown={(e) => { if (e.key === 'Enter') { if (filtered.length === 1) dest = filtered[0].path; else run() } }} />
  <div class="targets" role="listbox" aria-label={t('move.where')}>
    {#each filtered as x (x.path)}
      <button type="button" role="option" aria-selected={dest === x.path} class="target" class:on={dest === x.path} disabled={busy} onclick={() => (dest = x.path)} ondblclick={() => { dest = x.path; run() }}>
        {@html icons.folder()}<span>{x.label}</span>
      </button>
    {:else}
      <div class="hint">{t('move.none')}</div>
    {/each}
  </div>
  <div class="route">
    <span class="k">{t('move.from')}</span><code>{fromLabel}</code>
    <span class="k">{t('move.to')}</span><code class:empty={!dest}>{dest ? labelOf(dest) : t('move.pick')}</code>
  </div>
  {#if n.kind === 'project'}<p class="hint">{t('move.keeps')}</p>{/if}
</div>
<div class="ft">
  <button onclick={() => (store.dialog = null)} disabled={busy}>{t('dlg.cancel')}</button>
  <button class="p" bind:this={confirmBtn} disabled={!dest || busy} onclick={run}>{busy ? t('move.working') : t('move.go')}</button>
</div>
