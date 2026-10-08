<script lang="ts">
  import { store, ui } from '../lib/state.svelte'
  import { t, tn } from '../lib/i18n/index.svelte'
  import { icons } from '../lib/icons'
  import LauncherIcon from './LauncherIcon.svelte'
  import { nodeMenu, addMenu, openCtx } from '../lib/menus'
  import { dragSource, dropTarget } from '../lib/drag'
  import { lib, errMessage, type Node } from '../lib/api'

  let width = $state(window.innerWidth)

  // how many columns fit next to the card (min 480px); always at least 2
  let fit = $derived(Math.max(2, Math.floor((width / ui.zoom - 480 - 28) / 220)))
  let start = $derived(Math.max(0, store.path.length + 1 - fit))

  type Column = { index: number; node: Node; label: string }
  let columns = $derived.by(() => {
    const cols: Column[] = []
    for (let i = start; i <= store.path.length; i++) {
      const node = store.nodeAt(store.path.slice(0, i))
      if (!node) break
      cols.push({ index: i, node, label: i === 0 ? store.rootLabel : store.path[i - 1] })
    }
    return cols
  })

  function clickDir(i: number, n: Node) {
    store.go([...store.path.slice(0, i), n.name])
  }
  function clickProject(i: number, n: Node) {
    store.go(store.path.slice(0, i), n.name)
  }
  function moreClick(e: MouseEvent, n: Node) {
    e.stopPropagation()
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
    openCtx({ x: r.left, y: r.bottom + 4 }, nodeMenu(n))
  }
  function addClick(e: MouseEvent, n: Node) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
    openCtx({ x: r.left, y: r.bottom + 4 }, addMenu(n))
  }
  async function createMissing(n: Node) {
    try {
      await lib.CreateRoot(n.path)
      store.setTree(await lib.Rescan())
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }
</script>

<svelte:window bind:innerWidth={width} />

<div class="cols">
  {#if start > 0}
    {@const back = [store.rootLabel, ...store.path][start - 1]}
    <button class="hidden-cols" title={t('col.backTo', { name: back })} onclick={() => store.go(store.path.slice(0, start - 1))}>
      {tn('col.hidden', start)}
    </button>
  {/if}
  {#each columns as col (col.index + ':' + col.node.path)}
    {@const kids = (col.node.children ?? []).filter((c): c is Node => !!c)}
    <div class="col" role="group" aria-label={col.label}>
      <div class="colhead" class:drop={store.dropTarget === col.node.path} {...col.node.kind !== 'root' && !col.node.missing ? dropTarget(col.node.path) : {}}>
        <span title={col.node.path}>{col.label}</span>
        {#if col.node.kind !== 'root' && !col.node.missing}
          <button title={t('col.add')} aria-label={t('col.add')} onclick={(e) => addClick(e, col.node)}>+</button>
        {/if}
      </div>
      {#each kids as c (c.path)}
        {#if c.kind === 'project'}
          {@const on = col.index === store.path.length && store.sel === c.name}
          {@const ln = store.launcherFor(c).launcher}
          <button class="row" class:on title={c.desc || c.name} class:dragging={store.dragging === c.path} {...dragSource(c.path)}
            onclick={() => clickProject(col.index, c)}
            ondblclick={() => store.open(c)}
            onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); store.open(c) } }}
            oncontextmenu={(e) => { e.preventDefault(); clickProject(col.index, c); openCtx(e, nodeMenu(c)) }}>
            {#if ln}<LauncherIcon launcher={ln} />{:else}<span class="lico none"></span>{/if}
            <span class="name">{c.name}</span>
            {#if store.dirty[c.path]}<span class="gd" title={tn('card.modified', store.dirty[c.path])}></span>{/if}
            <span class="more" role="button" tabindex="-1" aria-label={t('col.actions')} onclick={(e) => moreClick(e, c)} onkeydown={() => {}}>⋯</span>
          </button>
        {:else}
          {@const on = store.path[col.index] === c.name}
          <button class="row" class:on title={c.path} class:dragging={store.dragging === c.path} class:drop={store.dropTarget === c.path}
            {...c.missing ? {} : { ...dragSource(c.path), ...dropTarget(c.path, () => clickDir(col.index, c)) }}
            onclick={() => clickDir(col.index, c)}
            oncontextmenu={(e) => { e.preventDefault(); openCtx(e, nodeMenu(c)) }}>
            {@html icons.folder()}
            <span class="name">{c.name}</span>
            {#if c.kind !== 'root'}<span class="n">{c.count}</span>{/if}
            {#if c.kind !== 'root'}<span class="more" role="button" tabindex="-1" aria-label={t('col.actions')} onclick={(e) => moreClick(e, c)} onkeydown={() => {}}>⋯</span>{/if}
            <span class="chev">›</span>
          </button>
        {/if}
      {/each}
      {#if !kids.length}
        <div class="colempty">
          {#if col.node.missing}
            {t('col.missing')}
            <button onclick={() => createMissing(col.node)}>{t('col.create')}</button>
          {:else if col.node.kind !== 'root'}
            {t('col.empty')}
            <button onclick={() => (store.dialog = { kind: 'newFolder', parent: col.node.path })}>{t('col.newFolder')}</button>
            <button onclick={() => (store.dialog = { kind: 'clone', parent: col.node.path })}>{t('col.cloneHere')}</button>
          {/if}
        </div>
      {/if}
    </div>
  {/each}
</div>
