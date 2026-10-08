<script lang="ts">
  import { store, isSpotlight } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { icons } from '../lib/icons'
  import { dropTarget } from '../lib/drag'
  import logo from '../assets/logo.svg'
  import WindowControls from './WindowControls.svelte'

  let crumbs = $derived([store.rootLabel, ...store.path])
</script>

<!-- three columns: logo and path | search, always centered | window buttons. The empty space (.drag) moves the window. -->
<header class="top" class:titlebar={store.st?.customTitleBar && !isSpotlight}>
  <div class="hl">
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
    <div class="drag"></div>
  </div>
  <button class="search-btn" onclick={() => (store.paletteOpen = true)}>
    {@html icons.search()}<span>{t('app.search')}</span><kbd>{store.mod} K</kbd>
  </button>
  <div class="hr">
    <div class="drag"></div>
    {#if store.st?.customTitleBar && !isSpotlight}<WindowControls />{/if}
  </div>
</header>
