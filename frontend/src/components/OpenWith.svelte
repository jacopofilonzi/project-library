<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import type { Node } from '../lib/api'
  import LauncherIcon from './LauncherIcon.svelte'

  let { node }: { node: Node } = $props()
  let open = $state(false)
  let wrap: HTMLDivElement | undefined = $state()

  let ls = $derived(store.enabledLaunchers)
  let def = $derived(store.defaultLauncher)
  let others = $derived(ls.filter((l) => l !== def))

  function launch(id: string) {
    open = false
    store.open(node, id)
  }
</script>

<svelte:window onmousedown={(e) => { if (open && wrap && !wrap.contains(e.target as Element)) open = false }} onkeydown={(e) => { if (e.key === 'Escape' && open) { e.stopPropagation(); open = false } }} />

<div class="open" class:split={others.length > 0} bind:this={wrap}>
  {#if !def}
    <button class="main" onclick={() => store.openSettings('launchers')}>{t('open.configureOne')}</button>
  {:else}
    <button class="main" onclick={() => launch(def.id)} title={store.launcherStatus[def.id] || t('open.notFound')}>
      <LauncherIcon launcher={def} />{t('open.with', { name: def.name })} <kbd>↵</kbd>
    </button>
    {#if others.length}
      <button class="caret" aria-label={t('open.others')} aria-expanded={open} onclick={() => (open = !open)}>▾</button>
      {#if open}
        <div class="menu" role="menu">
          <div class="h">{t('open.menuTitle', { name: node.name })}</div>
          {#each others as l (l.id)}
            <button role="menuitem" onclick={() => launch(l.id)}>
              <LauncherIcon launcher={l} /><span>{l.name}</span>
              <small>{store.launcherStatus[l.id] ? `${store.mod} ${ls.indexOf(l) + 1}` : t('open.notFound')}</small>
            </button>
          {/each}
          <hr />
          <button role="menuitem" onclick={() => { open = false; store.openSettings('launchers') }}><span>{t('open.configure')}</span></button>
        </div>
      {/if}
    {/if}
  {/if}
</div>
