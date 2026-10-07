<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t, tn, ago } from '../lib/i18n/index.svelte'
  import { icons } from '../lib/icons'
  import { lib, type GitInfo } from '../lib/api'

  let { info }: { info: GitInfo } = $props()

  function openRemote() {
    if (info.remoteWeb) lib.OpenURL(info.remoteWeb).then(() => store.toast(t('card.linkOpened')))
  }
</script>

{#if !store.st?.gitAvailable}
  <div class="nogit">{@html icons.branch()} {t('card.gitUnavailable')}</div>
{:else}
  <div class="git">
    <div class="top">
      <span class="pill mono">{@html icons.branch()}{info.branch || '—'}{info.detached ? ` (${t('card.detached')})` : ''}</span>
      {#if info.dirty}
        <span class="pill warn">{tn('card.modified', info.dirty)}</span>
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
  </div>
{/if}
