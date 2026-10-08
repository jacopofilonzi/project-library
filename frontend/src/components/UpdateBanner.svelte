<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { lib } from '../lib/api'
  import { icons } from '../lib/icons'

  // new release available: small banner at the bottom, until it is dismissed (it comes back at the next start)
  let u = $derived(store.update)
  let shown = $derived(!!u?.available && !store.updateDismissed)
  // toasts move up while the banner is visible, so they do not cover it
  $effect(() => {
    document.body.classList.toggle('has-upd', shown)
    return () => document.body.classList.remove('has-upd')
  })
</script>

{#if u && shown}
  <div class="upd" role="status">
    <span class="dot"></span>
    <span><b>Project Library {u.latest}</b> {t('update.available')}</span>
    <span class="v">{t('update.current', { version: u.current })}</span>
    <button class="notes" onclick={() => lib.OpenURL(u.notesUrl)}>{t('update.notes')}</button>
    <button class="go" onclick={() => lib.OpenURL(u.downloadUrl)}>{@html icons.clone(14)}{t('update.download')}</button>
    <button class="x" aria-label={t('update.dismiss')} title={t('update.dismiss')} onclick={() => (store.updateDismissed = true)}>✕</button>
  </div>
{/if}
