<script lang="ts">
  import { store, ui } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { icons } from '../lib/icons'
  import { dropTarget } from '../lib/drag'
  import logo from '../assets/logo.svg'

  let crumbs = $derived([store.rootLabel, ...store.path])

  function toggleTheme() {
    store.save((c) => (c.theme = ui.theme === 'dark' ? 'light' : 'dark'))
  }
</script>

<header class="top">
  <div class="logo"><img src={logo} alt="" width="22" height="22" /><span class="wm">project<span>/</span>library</span></div>
  <nav class="crumbs" aria-label="Path">
    {#each crumbs as c, i}
      {#if i > 0}<i>/</i>{/if}
      {@const target = store.nodeAt(store.path.slice(0, i))}
      <button class:drop={!!target && store.dropTarget === target.path}
        {...target && target.kind !== 'root' ? dropTarget(target.path) : {}}
        onclick={() => store.go(store.path.slice(0, i))}>{c}</button>
    {/each}
  </nav>
  <button class="search-btn" onclick={() => (store.paletteOpen = true)}>
    {@html icons.search()}<span>{t('app.search')}</span><kbd>{store.mod} K</kbd>
  </button>
  <button class="tbtn" onclick={() => (store.dialog = { kind: 'clone', parent: null })} disabled={!store.cfg.roots.length}>
    {@html icons.clone()}{t('app.clone')}
  </button>
  <div class="hgroup">
    <button class="ibtn" onclick={toggleTheme} title={ui.theme === 'dark' ? t('app.theme.toLight') : t('app.theme.toDark')} aria-label={ui.theme === 'dark' ? t('app.theme.toLight') : t('app.theme.toDark')}>
      {@html ui.theme === 'dark' ? icons.sun() : icons.moon()}
    </button>
    <button class="ibtn" onclick={() => store.openSettings()} title="{t('app.settings')} ({store.mod} P)" aria-label={t('app.settings')}>
      {@html icons.gear()}
    </button>
  </div>
</header>
