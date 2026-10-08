<script lang="ts">
  import { lib, type LanguageStat } from '../lib/api'
  import { t } from '../lib/i18n/index.svelte'

  // composizione dei linguaggi del progetto, come la barra di GitHub
  let { path }: { path: string } = $props()
  let stats = $state<LanguageStat[] | null>(null)
  let partial = $state(false)

  $effect(() => {
    const p = path
    stats = null
    lib.Languages(p).then((r) => {
      if (p !== path) return
      stats = (r.stats ?? []) as LanguageStat[]
      partial = r.partial
    }).catch(() => (stats = []))
  })

  const pct = (n: number) => (n >= 10 ? n.toFixed(0) : n.toFixed(1)) + '%'
</script>

{#if stats && stats.length}
  <div class="langs" title={partial ? t('card.langPartial') : ''}>
    <div class="lbar">
      {#each stats as s (s.name)}<span style="width:{s.percent}%;background:{s.color}" title="{s.name} {pct(s.percent)}"></span>{/each}
    </div>
    <ul>
      {#each stats as s (s.name)}
        <li><span class="dot" style="background:{s.color}"></span><b>{s.name === 'Other' ? t('card.langOther') : s.name}</b> {pct(s.percent)}</li>
      {/each}
      {#if partial}<li class="part">{t('card.langPartialShort')}</li>{/if}
    </ul>
  </div>
{/if}
