<script lang="ts">
  import { Browser } from '@wailsio/runtime'
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { renderMarkdown } from '../lib/markdown'
  import type { ReadmeResult } from '../lib/api'

  let { readme, dir, error }: { readme: ReadmeResult | null; dir: string; error: boolean } = $props()

  let html = $derived(readme?.found && readme.format === 'markdown' ? renderMarkdown(readme.content, dir) : '')

  // links open in the browser; relative links and anchors do not
  function onclick(e: MouseEvent) {
    const a = (e.target as Element).closest('a')
    if (!a) return
    e.preventDefault()
    const href = a.getAttribute('href') ?? ''
    if (/^https?:\/\//i.test(href)) {
      Browser.OpenURL(href)
      store.toast(t('card.linkOpened'))
    } else if (href.startsWith('#')) {
      const id = decodeURIComponent(href.slice(1))
      ;(e.currentTarget as HTMLElement).querySelector(`[id="${CSS.escape(id)}"]`)?.scrollIntoView({ behavior: 'smooth' })
    } else if (href) {
      store.toast(t('card.relativeLink'))
    }
  }
</script>

<div class="readme">
  <div class="bar"><span>{readme?.found ? readme.name : 'README'}</span></div>
  {#if error}
    <div class="placeholder">{t('card.readmeError')}</div>
  {:else if !readme}
    <div class="placeholder">{t('card.readmeLoading')}</div>
  {:else if !readme.found}
    <div class="placeholder">{t('card.noReadme')}</div>
  {:else if readme.format === 'markdown'}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="md" {onclick}>{@html html}</div>
  {:else}
    <pre class="md text">{readme.content}</pre>
  {/if}
  {#if readme?.truncated}<div class="placeholder">{t('card.readmeTruncated')}</div>{/if}
</div>
