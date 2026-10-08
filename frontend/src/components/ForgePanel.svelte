<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t, tn } from '../lib/i18n/index.svelte'
  import { lib, type ForgeInfo } from '../lib/api'

  // pull/merge request, issue e CI del remote, tramite gh o glab; nulla se nessuna CLI ha un account su quell'host
  let { remote, branch }: { remote: string; branch: string } = $props()

  let info = $state<ForgeInfo | null>(null)
  let showPrs = $state(false)

  $effect(() => {
    const r = remote
    const b = branch
    info = null
    showPrs = false
    lib.ForgeInfo(r, b).then((i) => { if (r === remote && b === branch) info = i }).catch(() => {})
  })

  let gitlab = $derived(String(info?.kind) === 'gitlab')
  let prLabel = $derived.by(() => {
    if (!info) return ''
    if (!info.prCount) return t(gitlab ? 'forge.noMrs' : 'forge.noPrs')
    const label = tn(gitlab ? 'forge.mrs' : 'forge.prs', info.prCount)
    return info.prMore ? label.replace(String(info.prCount), info.prCount + '+') : label
  })
  const ciClass: Record<string, string> = { success: 'ok', failure: 'bad', running: 'warn', pending: 'warn' }

  function open(url: string) {
    lib.OpenURL(url).then(() => store.toast(t('card.linkOpened')))
  }
  const issuesURL = (i: ForgeInfo) => i.web + (String(i.kind) === 'gitlab' ? '/-/issues' : '/issues')
</script>

{#if info}
  <div class="forge">
    {#if info.prCount}
      <button class="pill act" aria-expanded={showPrs} onclick={() => (showPrs = !showPrs)}>{prLabel} {showPrs ? '▴' : '▾'}</button>
    {:else}
      <span class="pill">{prLabel}</span>
    {/if}
    {#if info.issues >= 0}
      <button class="pill act" title={issuesURL(info)} onclick={() => open(issuesURL(info!))}>{tn('forge.issues', info.issues)} ↗</button>
    {/if}
    {#if info.ci}
      {@const ci = info.ci}
      <button class="pill act {ciClass[ci.state] ?? ''}" title={ci.runs > 1 ? `${ci.name} · ${t('forge.workflows', { n: ci.runs })}` : ci.name} onclick={() => open(ci.url)}>{t('forge.ci.' + ci.state)} ↗</button>
    {/if}
  </div>
  {#if showPrs}
    <div class="prs">
      {#each info.prs ?? [] as pr (pr.number)}
        <button class="pr" onclick={() => open(pr.url)} title={pr.url}>
          <code>#{pr.number}</code><span class="pt">{pr.title}</span>{#if pr.draft}<span class="tg">{t('forge.draft')}</span>{/if}<span class="pa">{pr.author}</span>
        </button>
      {/each}
    </div>
  {/if}
{/if}
