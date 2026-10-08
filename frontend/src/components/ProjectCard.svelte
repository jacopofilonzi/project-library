<script lang="ts">
  import { untrack, tick } from 'svelte'
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { icons } from '../lib/icons'
  import LangBar from './LangBar.svelte'
  import { lib, errMessage, type Node, type GitInfo, type ReadmeResult } from '../lib/api'
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
      /* git errors: the card stays without the panel */
    }
  }

  // project without git: git init in place
  async function initGit() {
    try {
      await lib.InitProject(node.path, true, false)
      store.setTree(await lib.Tree())
      await tick()
      loadGit(node.path)
      store.toast(t('init.gitDone', { name: node.name }))
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }

  // reload only when the project changes (untrack: config changes must not reload the README)
  $effect(() => {
    const p = node.path
    untrack(() => load(p))
  })
  // refresh the git state after an action that changes it (e.g. publishing)
  $effect(() => {
    if (store.gitChanged) untrack(() => loadGit(node.path))
  })
  // refresh the git state when the window comes back to the front
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
        <LangBar path={node.path} />
      </div>
      <div class="acts">
        <OpenWith {node} />
        <button class="ibtn" title={t('card.explorer', { key: store.mod + ' E' })} aria-label={t('ctx.reveal')} onclick={() => store.reveal(node.path)}>{@html icons.folder(18)}</button>
      </div>
    </div>
    {#if store.cfg.gitInfo}
      {#if node.hasGit}
        {#if git}<GitPanel info={git} path={node.path} onrefresh={() => loadGit(node.path)} />{/if}
      {:else}
        <div class="nogit">{@html icons.branch()} {t('card.notGit')}{#if store.st?.gitAvailable}<button class="btn" onclick={initGit}>{t('init.gitButton')}</button>{/if}</div>
      {/if}
    {/if}
    <ReadmeView {readme} dir={node.path} error={readmeError} />
  </div>
</section>
