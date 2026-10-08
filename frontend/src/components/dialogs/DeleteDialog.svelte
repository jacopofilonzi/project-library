<script lang="ts">
  import { untrack } from 'svelte'
  import { store } from '../../lib/state.svelte'
  import { t, tn } from '../../lib/i18n/index.svelte'
  import { lib, type Node } from '../../lib/api'

  let { node }: { node: Node } = $props()

  let warnings = $state<string[]>([])
  let strong = $state(false) // strong confirmation: the name must be typed
  let loading = $state(untrack(() => node.kind === 'project'))
  let typed = $state('')
  let nameInput: HTMLInputElement | undefined = $state()
  let cancelBtn: HTMLButtonElement | undefined = $state()
  // focus on the name field if strong confirmation is needed, otherwise on Cancel (never on Delete)
  $effect(() => {
    if (loading) return
    ;(strong ? nameInput : cancelBtn)?.focus()
  })
  let busy = $state(false)

  // for projects: check uncommitted changes, unpushed commits, missing remote
  $effect(() => {
    if (node.kind !== 'project') return
    ;(async () => {
      const w: string[] = []
      if (node.hasGit && store.st?.gitAvailable) {
        try {
          const g = await lib.GitInfo(node.path)
          if (g.dirty) w.push(t('dlg.warnDirty', { n: g.dirty }))
          if (g.ahead) w.push(t('dlg.warnAhead', { n: g.ahead }))
          if (!g.remote) w.push(t('dlg.warnNoRemote'))
        } catch {
          w.push(t('dlg.warnNoGit'))
        }
      } else if (!node.hasGit) {
        w.push(t('dlg.warnNoGit'))
      }
      warnings = w
      strong = w.length > 0
      loading = false
    })()
  })

  let dirs = $derived((node.children ?? []).filter((c) => c && c.kind !== 'project').length)
  let ok = $derived(!loading && !busy && (!strong || typed === node.name))

  async function confirm() {
    if (!ok) return
    busy = true
    await store.trash(node)
    store.setTree(await lib.Tree())
    store.dialog = null
  }
</script>

<h3>{t('dlg.deleteTitle', { name: node.name })}</h3>
<div class="bd">
  {#if node.kind === 'project'}
    <p>{t('dlg.deleteProject')}</p>
    {#if warnings.length}<div class="warn">{t('dlg.warnings', { list: warnings.join(' · ') })}</div>{/if}
  {:else}
    <p>
      {#if dirs}
        {t('dlg.deleteDir', { projects: tn('dlg.nProjects', node.count), dirs: tn('dlg.nDirs', dirs) })}
      {:else}
        {t('dlg.deleteDirOnlyProjects', { projects: tn('dlg.nProjects', node.count) })}
      {/if}
    </p>
    <div class="warn">{t('dlg.deleteDirWarn')}</div>
  {/if}
  <div class="final selectable">{node.path}</div>
  {#if strong}
    <label>{t('dlg.typeName', { name: node.name })}<input type="text" bind:this={nameInput} bind:value={typed} autocomplete="off" spellcheck="false" onkeydown={(e) => e.key === 'Enter' && confirm()} /></label>
  {/if}
  <p class="hint">{t('dlg.toTrash', { trash: store.trashName })}</p>
</div>
<div class="ft">
  <button bind:this={cancelBtn} onclick={() => (store.dialog = null)}>{t('dlg.cancel')}</button>
  <button class="d" disabled={!ok} onclick={confirm}>{t('dlg.delete')}</button>
</div>
