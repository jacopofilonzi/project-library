<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import type { Node } from '../lib/api'
  import LauncherIcon from './LauncherIcon.svelte'

  let { node }: { node: Node } = $props()
  let open = $state(false)
  let wrap: HTMLDivElement | undefined = $state()

  let ls = $derived(store.enabledLaunchers)
  let choice = $derived(store.launcherFor(node))
  let def = $derived(choice.launcher)
  let others = $derived(ls.filter((l) => l !== def))
  let forced = $derived(store.cfg.projectLaunchers[node.path] ?? '')

  function launch(id: string) {
    open = false
    store.open(node, id)
  }
  function force(id: string) {
    open = false
    if (id !== forced) store.setProjectLauncher(node, id)
  }
</script>

<svelte:window onmousedown={(e) => { if (open && wrap && !wrap.contains(e.target as Element)) open = false }} onkeydown={(e) => { if (e.key === 'Escape' && open) { e.stopPropagation(); open = false } }} />

<div class="open" class:split={others.length > 0} bind:this={wrap}>
  {#if !def}
    <button class="main" onclick={() => store.openSettings('launchers')}>{t('open.configureOne')}</button>
  {:else}
    <button class="main" onclick={() => launch(def.id)} title={`${t('open.reason.' + choice.reason)} · ${store.launcherStatus[def.id] || t('open.notFound')}`}>
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
          <div class="h">{t('open.forProject')}</div>
          <button role="menuitemradio" aria-checked={!forced} onclick={() => force('')}>
            <span>{t('open.auto', { name: store.autoLauncherFor(node).launcher?.name ?? '—' })}</span><small>{forced ? '' : '✓'}</small>
          </button>
          {#each ls as l (l.id)}
            <button role="menuitemradio" aria-checked={forced === l.id} onclick={() => force(l.id)}>
              <LauncherIcon launcher={l} /><span>{t('open.always', { name: l.name })}</span><small>{forced === l.id ? '✓' : ''}</small>
            </button>
          {/each}
          <hr />
          <button role="menuitem" onclick={() => { open = false; store.openSettings('launchers') }}><span>{t('open.configure')}</span></button>
        </div>
      {/if}
    {/if}
  {/if}
</div>
