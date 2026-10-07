<script lang="ts">
  import { store } from '../../lib/state.svelte'
  import { t } from '../../lib/i18n/index.svelte'
  import { lib, errMessage } from '../../lib/api'

  // mode "new": crea una cartella in parent; mode "rename": rinomina path
  let { mode, parent = '', path = '', name: initial = '' }: { mode: 'new' | 'rename'; parent?: string; path?: string; name?: string } = $props()

  let name = $state(initial)
  let problem = $state('')
  let busy = $state(false)
  let input: HTMLInputElement | undefined = $state()

  const dir = mode === 'new' ? parent : path.slice(0, Math.max(path.lastIndexOf('\\'), path.lastIndexOf('/')))

  $effect(() => {
    input?.focus()
    if (mode === 'rename') input?.select()
  })

  // verifica il nome mentre si scrive (validazione fatta dal backend, regole del sistema operativo)
  let timer: ReturnType<typeof setTimeout>
  $effect(() => {
    const n = name
    clearTimeout(timer)
    if (!n || (mode === 'rename' && n === initial)) { problem = ''; return }
    timer = setTimeout(async () => {
      const code = await lib.CheckName(dir, n)
      // rinomina che cambia solo maiuscole/minuscole: consentita
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
        const renamed = await lib.Rename(path, name)
        const wasSelected = store.selected?.path === path
        store.setTree(await lib.Tree())
        if (wasSelected || store.current?.path === renamed) store.goToPath(renamed)
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
