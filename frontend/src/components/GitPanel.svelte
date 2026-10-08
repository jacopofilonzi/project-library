<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t, tn, ago } from '../lib/i18n/index.svelte'
  import { icons } from '../lib/icons'
  import { lib, errMessage, serviceName, type GitInfo } from '../lib/api'
  import ForgePanel from './ForgePanel.svelte'

  let { info, path, onrefresh }: { info: GitInfo; path: string; onrefresh: () => void } = $props()

  let busy = $state<'' | 'fetch' | 'pull'>('')
  let showChanges = $state(false)
  let changes = $state<{ status: string; path: string }[] | null>(null)

  function openRemote() {
    if (info.remoteWeb) lib.OpenURL(info.remoteWeb).then(() => store.toast(t('card.linkOpened')))
  }

  async function fetchNow() {
    busy = 'fetch'
    try {
      await lib.GitFetch(path)
      store.toast(t('git.fetched'))
      onrefresh()
    } catch (e) {
      store.toast(errMessage(e), true)
    }
    busy = ''
  }

  async function pull() {
    busy = 'pull'
    try {
      const out = await lib.GitPull(path)
      store.toast(/up to date/i.test(out) ? t('git.upToDate') : t('git.pulled'))
      onrefresh()
      if (showChanges) loadChanges()
    } catch (e) {
      store.toast(errMessage(e), true)
    }
    busy = ''
  }

  async function loadChanges() {
    changes = ((await lib.GitChanges(path).catch(() => [])) ?? []) as { status: string; path: string }[]
  }

  function toggleChanges() {
    showChanges = !showChanges
    if (showChanges) loadChanges()
  }

  // publish on GitHub/GitLab: needs a logged-in account and a repository without a remote
  let publishLabel = $derived.by(() => {
    const kinds = new Set(store.forgeAccounts.map((a) => a.kind))
    return t('forge.publishOn', { service: kinds.size === 1 ? serviceName([...kinds][0]) : 'GitHub / GitLab' })
  })
  function publish() {
    const node = store.index.get(path)?.node
    if (node) store.dialog = { kind: 'publish', node }
  }

  // legend of the git status codes
  const statusLabel = (s: string) => t('git.status.' + (s === '??' ? 'untracked' : s[0] ?? 'M'))
</script>

{#if !store.st?.gitAvailable}
  <div class="nogit">{@html icons.branch()} {t('card.gitUnavailable')}</div>
{:else}
  <div class="git">
    <div class="top">
      <span class="pill mono">{@html icons.branch()}{info.branch || '—'}{info.detached ? ` (${t('card.detached')})` : ''}</span>
      {#if info.dirty}
        <button class="pill warn act" aria-expanded={showChanges} onclick={toggleChanges}>{tn('card.modified', info.dirty)} {showChanges ? '▴' : '▾'}</button>
      {:else}
        <span class="pill ok">{t('card.clean')}</span>
      {/if}
      {#if info.remote && info.hasUpstream}
        {#if info.ahead}<span class="pill warn">{t('card.ahead', { n: info.ahead })}</span>{/if}
        {#if info.behind}<span class="pill warn">{t('card.behind', { n: info.behind })}</span>{/if}
        {#if !info.ahead && !info.behind}<span class="pill ok">{t('card.synced')}</span>{/if}
      {:else if info.remote}
        <span class="pill">{t('card.noUpstream')}</span>
      {/if}
      {#if info.remote}
        {#if info.remoteWeb}
          <button class="remote" onclick={openRemote} title={info.remote}>{info.remoteWeb.replace(/^https:\/\//, '')} ↗</button>
        {:else}
          <span class="remote none" title={info.remote}>{info.remote}</span>
        {/if}
      {:else}
        <span class="remote none">{t('card.noRemote')}</span>
      {/if}
    </div>
    <div class="commit">
      {#if info.commit}
        <code>{info.commit.hash}</code>
        <span class="msg selectable" title={info.commit.subject}>{info.commit.subject}</span>
        <span class="who">{info.commit.author} · {ago(info.commit.time * 1000)}</span>
      {:else}
        <span class="who">{t('card.noCommits')}</span>
      {/if}
    </div>
    {#if info.remote}
      <div class="gitacts">
        <button class="btn" disabled={!!busy} onclick={fetchNow}>{busy === 'fetch' ? t('git.fetching') : t('git.fetch')}</button>
        <button class="btn" disabled={!!busy || !info.hasUpstream} title={info.hasUpstream ? t('git.pullHint') : t('card.noUpstream')} onclick={pull}>{busy === 'pull' ? t('git.pulling') : t('git.pull')}</button>
      </div>
      {#if store.forgeAccounts.length}<ForgePanel remote={info.remote} branch={info.branch} />{/if}
    {:else if store.forgeAccounts.length}
      <div class="gitacts"><button class="btn" onclick={publish}>{publishLabel}</button></div>
    {/if}
    {#if showChanges}
      <div class="changes">
        {#if changes === null}
          <div class="who">…</div>
        {:else}
          {#each changes as c (c.path)}
            <div class="ch"><span class="st st-{c.status === '??' ? 'u' : c.status[0]}" title={statusLabel(c.status)}>{c.status}</span><span class="selectable">{c.path}</span></div>
          {/each}
          {#if changes.length >= 200}<div class="who">{t('git.truncated')}</div>{/if}
        {/if}
      </div>
    {/if}
  </div>
{/if}
