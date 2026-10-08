<script lang="ts">
  import { onMount } from 'svelte'
  import { Events, Window } from '@wailsio/runtime'
  import { t } from '../lib/i18n/index.svelte'

  // Minimize / maximize / close of the custom title bar (Windows). The --wails-non-client-region
  // property tells Windows that these are the caption buttons: that is what shows the snap layouts
  // flyout over maximize. Wails forwards the click to the page, so the actions run here.
  let maximised = $state(false)

  onMount(() => {
    const sync = () => Window.IsMaximised().then((m) => (maximised = m)).catch(() => {})
    sync()
    const offs = ['common:WindowMaximise', 'common:WindowUnMaximise', 'common:WindowRestore'].map((e) => Events.On(e, sync))
    window.addEventListener('resize', sync)
    return () => {
      offs.forEach((off) => off())
      window.removeEventListener('resize', sync)
    }
  })
</script>

<div class="wctl">
  <button class="min" aria-label={t('window.minimize')} title={t('window.minimize')} onclick={() => Window.Minimise()}>
    <svg viewBox="0 0 10 10" aria-hidden="true"><path d="M0 5.5h10" stroke="currentColor" stroke-width="1" /></svg>
  </button>
  <button class="max" aria-label={maximised ? t('window.restore') : t('window.maximize')} title={maximised ? t('window.restore') : t('window.maximize')} onclick={() => Window.ToggleMaximise()}>
    {#if maximised}
      <svg viewBox="0 0 10 10" aria-hidden="true"><rect x=".5" y="2.5" width="7" height="7" fill="none" stroke="currentColor" stroke-width="1" /><path d="M2.5 2.5V.5h7v7h-2" fill="none" stroke="currentColor" stroke-width="1" /></svg>
    {:else}
      <svg viewBox="0 0 10 10" aria-hidden="true"><rect x=".5" y=".5" width="9" height="9" fill="none" stroke="currentColor" stroke-width="1" /></svg>
    {/if}
  </button>
  <button class="close" aria-label={t('window.close')} title={t('window.close')} onclick={() => Window.Close()}>
    <svg viewBox="0 0 10 10" aria-hidden="true"><path d="M0 0l10 10M10 0L0 10" stroke="currentColor" stroke-width="1" /></svg>
  </button>
</div>
