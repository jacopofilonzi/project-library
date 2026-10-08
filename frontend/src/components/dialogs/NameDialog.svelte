<script lang="ts">
  import { untrack } from 'svelte'
  import { store } from '../../lib/state.svelte'
  import { t } from '../../lib/i18n/index.svelte'
  import { lib, errMessage } from '../../lib/api'

  // mode "new": creates a folder in parent; mode "rename": renames path
  let { mode, parent = '', path = '', name: initial = '' }: { mode: 'new' | 'rename'; parent?: string; path?: string; name?: string } = $props()

  // the dialog is recreated at every opening: the props are only initial values
  let name = $state(untrack(() => initial))
  let problem = $state('')
  let busy = $state(false)
  let input: HTMLInputElement | undefined = $state()

  const dir = untrack(() => (mode === 'new' ? parent : path.slice(0, Math.max(path.lastIndexOf('\\'), path.lastIndexOf('/')))))

  $effect(() => {
    input?.focus()
    if (mode === 'rename') input?.select()
  })

  // checks the name while typing (validated by the backend, operating system rules)
  let timer: ReturnType<typeof setTimeout>
  $effect(() => {
    const n = name
    clearTimeout(timer)
    if (!n || (mode === 'rename' && n === initial)) { problem = ''; return }
    timer = setTimeout(async () => {
      const code = await lib.CheckName(dir, n)
      // rename that only changes the case: allowed
      problem = mode === 'rename' && code === 'exists' && n.toLowerCase() === initial.toLowerCase() ? '' : code
    }, 120)
  })

  let ok = $derived(!!name.trim() && !problem && !busy && !(mode === 'rename' && name === initial))

  async function submit() {
    if (!ok) return
    busy = true
    try {
      if (mode === 'new') {
        const created = await lib.Mkdir(parent, name)
        store.setTree(await lib.Tree())
        store.goToPath(created)
        store.toast(t('dlg.created', { name: name.trim() }))
      } else {
        // if you were looking at the renamed item (or something inside it) you stay there
        const wasHere = store.selected?.path === path || store.current?.path === path || !!store.current?.path.startsWith(path + (path.includes('\\') ? '\\' : '/'))
        const renamed = await lib.Rename(path, name)
        store.setTree(await lib.Tree())
        if (wasHere) store.goToPath(renamed)
        store.toast(t('dlg.renamed', { name: name.trim() }))
      }
      store.dialog = null
    } catch (e) {
      store.toast(errMessage(e), true)
      busy = false
    }
  }
</script>

<h3>{mode === 'new' ? t('dlg.newFolder') : t('dlg.renameTitle', { name: initial })}</h3>
<div class="bd">
  <label>{t('dlg.name')}<input type="text" bind:this={input} bind:value={name} autocomplete="off" spellcheck="false" onkeydown={(e) => e.key === 'Enter' && submit()} /></label>
  {#if problem}
    <div class="hint err">{t(problem.startsWith('name.') ? problem : 'name.' + problem)}</div>
  {:else if mode === 'new'}
    <div class="hint">{t('dlg.willCreate', { path: parent })}</div>
  {/if}
</div>
<div class="ft">
  <button onclick={() => (store.dialog = null)}>{t('dlg.cancel')}</button>
  <button class="p" disabled={!ok} onclick={submit}>{mode === 'new' ? t('dlg.create') : t('dlg.rename')}</button>
</div>
