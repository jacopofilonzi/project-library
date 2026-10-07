<script lang="ts">
  import { untrack } from 'svelte'
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { icons, colorOf } from '../lib/icons'
  import { lib, type Node, type GitInfo, type ReadmeResult } from '../lib/api'
  import OpenWith from './OpenWith.svelte'
  import GitPanel from './GitPanel.svelte'
  import ReadmeView from './ReadmeView.svelte'

  let { node }: { node: Node } = $props()

  let readme = $state<ReadmeResult | null>(null)
  let readmeError = $state(false)
  let git = $state<GitInfo | null>(null)

  async function load(path: string) {
    readme = null
    readmeError = false
    git = null
    lib.Readme(path).then((r) => { if (node.path === path) readme = r }).catch(() => { if (node.path === path) readmeError = true })
    loadGit(path)
  }
  async function loadGit(path: string) {
    if (!store.cfg.gitInfo || !node.hasGit) return
    try {
      const info = await lib.GitInfo(path)
      if (node.path === path) git = info
    } catch {
      /* errori git: la scheda resta senza pannello */
    }
  }

  // ricarica solo quando cambia progetto (untrack: le modifiche alla config non devono ricaricare il README)
  $effect(() => {
    const p = node.path
    untrack(() => load(p))
  })
  // aggiorna lo stato git quando la finestra torna in primo piano
  $effect(() => {
    const f = () => loadGit(node.path)
    window.addEventListener('focus', f)
    return () => window.removeEventListener('focus', f)
  })
</script>

<section class="detail">
  <div class="in">
    <div class="head">
      <div>
        <div class="path selectable">{node.path}</div>
        <h1>{node.name}</h1>
        {#if node.lang}<span class="tag"><span class="dot" style="background:{colorOf(node.lang)}"></span>{node.lang}</span>{/if}
      </div>
      <div class="acts">
        <OpenWith {node} />
        <button class="ibtn" title={t('card.explorer', { key: store.mod + ' E' })} aria-label={t('ctx.reveal')} onclick={() => store.reveal(node.path)}>{@html icons.folder(18)}</button>
      </div>
    </div>
    {#if store.cfg.gitInfo}
      {#if node.hasGit}
        {#if git}<GitPanel info={git} />{/if}
      {:else}
        <div class="nogit">{@html icons.branch()} {t('card.notGit')}</div>
      {/if}
    {/if}
    <ReadmeView {readme} dir={node.path} error={readmeError} />
  </div>
</section>
