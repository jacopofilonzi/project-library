<script lang="ts">
  import { store } from '../../lib/state.svelte'
  import { t } from '../../lib/i18n/index.svelte'
  import { lib, errMessage } from '../../lib/api'

  let { path, name }: { path: string; name: string } = $props()

  const gitOk = !!store.st?.gitAvailable
  let gitInit = $state(gitOk)
  let readme = $state(true)
  let busy = $state(false)
  let okBtn: HTMLButtonElement | undefined = $state()
  $effect(() => okBtn?.focus())

  async function run() {
    if (busy || (!gitInit && !readme)) return
    busy = true
    try {
      await lib.InitProject(path, gitInit, readme)
      store.setTree(await lib.Tree())
      store.goToPath(path)
      store.dialog = null
      store.toast(t('init.done', { name }))
    } catch (e) {
      store.toast(errMessage(e), true)
      busy = false
    }
  }
</script>

<h3>{t('init.title', { name })}</h3>
<div class="bd">
  <p>{t('init.text')}</p>
  <label class="chk"><input type="checkbox" bind:checked={gitInit} disabled={!gitOk} /> {t('init.git')}</label>
  {#if !gitOk}<div class="hint">{t('dlg.noGit')}</div>{/if}
  <label class="chk"><input type="checkbox" bind:checked={readme} /> {t('init.readme', { name })}</label>
  <div class="final selectable">{path}</div>
</div>
<div class="ft">
  <button onclick={() => (store.dialog = null)}>{t('dlg.cancel')}</button>
  <button class="p" bind:this={okBtn} disabled={busy || (!gitInit && !readme)} onclick={run}>{t('init.run')}</button>
</div>
