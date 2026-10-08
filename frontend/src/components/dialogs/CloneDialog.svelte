<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import { store } from '../../lib/state.svelte'
  import { t } from '../../lib/i18n/index.svelte'
  import { lib, errMessage, errCode, type CloneSuggestion, type CloneProgress, type Repo } from '../../lib/api'

  // parent: folder it was opened from; null = from the navbar (github/<owner> is suggested)
  let { parent }: { parent: string | null } = $props()

  // the dialog is recreated at every opening: parent is only the initial value
  const from = untrack(() => parent)
  const roots = store.cfg.roots
  // reference root: the one that contains parent, otherwise the first
  const root = (from && roots.find((r) => from === r || from.startsWith(r + '\\') || from.startsWith(r + '/'))) || roots[0] || ''
  const sep = root.includes('\\') ? '\\' : '/'
  const relOf = (p: string) => (p.length > root.length ? p.slice(root.length + 1) : '').split(/[\\/]/).filter(Boolean).join('/')
  const startRel = from ? relOf(from) : relOf(store.current?.path ?? root)

  let url = $state('')
  let urlInput: HTMLInputElement | undefined = $state()
  let searchInput: HTMLInputElement | undefined = $state()

  // with gh or glab logged in you pick from your repositories, otherwise you paste the URL
  const hasForge = store.forgeAccounts.length > 0
  let tab = $state<'url' | 'repos'>(hasForge ? 'repos' : 'url')
  let repos = $state<Repo[] | null>(null)
  let repoErrors = $state<string[]>([])
  let q = $state('')
  async function loadRepos(refresh = false) {
    repos = null
    const r = await lib.ForgeRepos(refresh).catch(() => null)
    repos = r?.repos ?? []
    repoErrors = r?.errors ?? []
  }
  if (hasForge) loadRepos()
  let filtered = $derived.by(() => {
    const words = q.toLowerCase().split(/\s+/).filter(Boolean)
    return (repos ?? []).filter((r) => words.every((w) => (r.fullName + ' ' + r.description).toLowerCase().includes(w))).slice(0, 200)
  })
  function pick(r: Repo) {
    url = r.cloneUrl
  }

  // autofocus does not work on elements added after load: the focus is given by hand
  $effect(() => (tab === 'repos' ? searchInput : urlInput)?.focus())
  let dest = $state(startRel)
  let name = $state('')
  let openAfter = $state(true)
  let sug = $state<CloneSuggestion | null>(null)
  let nameTouched = false
  let destTouched = !!from
  let problem = $state('')
  let running = $state(false)
  let progress = $state<{ phase: string; percent: number } | null>(null)
  const id = 'c' + Date.now()

  const off = Events.On('clone:progress', (ev: { data: CloneProgress }) => {
    if (ev.data.id === id) progress = { phase: ev.data.phase, percent: ev.data.percent }
  })
  onDestroy(() => off())

  let suggestedDest = $derived(sug?.ok && sug.host === 'github.com' ? `github/${sug.owner}` : sug?.ok && sug.host === 'gitlab.com' ? `gitlab/${sug.owner}` : '')
  let parentAbs = $derived(root + (dest.trim() ? sep + dest.split('/').filter(Boolean).join(sep) : ''))
  let finalPath = $derived(parentAbs + (name ? sep + name : ''))

  let timer: ReturnType<typeof setTimeout>
  $effect(() => {
    const u = url
    clearTimeout(timer)
    timer = setTimeout(async () => {
      sug = u.trim() ? await lib.ParseRepoURL(u) : null
      if (sug?.ok) {
        if (!nameTouched) name = sug.repo
        if (!destTouched && suggestedDest) dest = suggestedDest
      }
    }, 150)
  })

  let missingDirs = $state(false)
  $effect(() => {
    const p = parentAbs
    const n = name
    ;(async () => {
      missingDirs = !(await lib.Exists(p))
      problem = n ? (missingDirs ? '' : await lib.CheckName(p, n)) : ''
    })()
  })

  let ok = $derived(!!sug?.ok && !!name.trim() && !problem && !running && !!store.st?.gitAvailable)
  let def = $derived(store.defaultLauncher)

  async function run() {
    if (!ok) return
    running = true
    progress = { phase: '', percent: 0 }
    try {
      const path = await lib.Clone(id, url.trim(), parentAbs, name.trim())
      store.setTree(await lib.Tree())
      store.goToPath(path)
      store.dialog = null
      const node = store.index.get(path)?.node
      if (openAfter && node && def) store.open(node)
      else store.toast(t('dlg.cloned', { path }))
    } catch (e) {
      running = false
      progress = null
      if (errCode(e) !== 'cancelled') store.toast(errMessage(e), true)
    }
  }

  // git clone phases, translated when we know them
  function phaseLabel(p: string) {
    const k = 'dlg.phase.' + p
    const tr = t(k)
    return tr === k ? p : tr
  }

  function cancel() {
    if (running) lib.CancelClone(id)
    store.dialog = null
  }
</script>

<h3>{t('dlg.clone')}</h3>
<div class="bd">
  {#if !store.st?.gitAvailable}<div class="warn">{t('dlg.noGit')}</div>{/if}
  {#if hasForge}
    <div class="tabs" role="tablist">
      <button role="tab" aria-selected={tab === 'repos'} class:on={tab === 'repos'} disabled={running} onclick={() => (tab = 'repos')}>{t('dlg.fromRepos')}</button>
      <button role="tab" aria-selected={tab === 'url'} class:on={tab === 'url'} disabled={running} onclick={() => (tab = 'url')}>{t('dlg.fromUrl')}</button>
    </div>
  {/if}
  {#if tab === 'repos'}
    <div class="repos-h">
      <input type="text" bind:this={searchInput} bind:value={q} placeholder={t('dlg.searchRepos')} autocomplete="off" spellcheck="false" disabled={running}
        onkeydown={(e) => { if (e.key === 'Enter') { if (filtered.length === 1) pick(filtered[0]); else run() } }} />
      <button class="lnk" type="button" disabled={running || repos === null} onclick={() => loadRepos(true)}>{t('dlg.reposRefresh')}</button>
    </div>
    <div class="repos" role="listbox" aria-label={t('dlg.fromRepos')}>
      {#if repos === null}
        <div class="hint">{t('dlg.reposLoading')}</div>
      {:else}
        {#each filtered as r (r.kind + r.host + r.fullName)}
          <button type="button" role="option" aria-selected={url === r.cloneUrl} class="repo" class:on={url === r.cloneUrl} disabled={running} onclick={() => pick(r)}>
            <span class="rn">{r.fullName}{#if r.private}<span class="tg">{t('dlg.private')}</span>{/if}</span>
            <span class="rh">{r.host}</span>
            {#if r.description}<span class="rd">{r.description}</span>{/if}
          </button>
        {:else}
          <div class="hint">{t('dlg.reposNone')}</div>
        {/each}
      {/if}
    </div>
    {#if repoErrors.length}<div class="hint err">{t('dlg.reposErrors', { list: repoErrors.join(' · ') })}</div>{/if}
  {:else}
    <label>{t('dlg.url')}
      <input type="text" bind:this={urlInput} bind:value={url} placeholder={t('dlg.urlPlaceholder')} autocomplete="off" spellcheck="false" disabled={running} onkeydown={(e) => e.key === 'Enter' && run()} />
    </label>
    {#if !hasForge}
      <div class="hint">{t('dlg.cliHint')} <button class="lnk" type="button" onclick={() => store.openSettings('git')}>{t('dlg.cliHintLink')}</button></div>
    {/if}
  {/if}
  <label>{t('dlg.dest')}
    <div class="pre"><span title={root}>{root}{sep}</span><input type="text" bind:value={dest} oninput={() => (destTouched = true)} autocomplete="off" spellcheck="false" disabled={running} /></div>
    {#if suggestedDest && dest !== suggestedDest}
      <span class="hint sug">{t('dlg.suggested')} <button type="button" onclick={() => (dest = suggestedDest)}>{suggestedDest}</button></span>
    {/if}
  </label>
  <label>{t('dlg.folderName')}<input type="text" bind:value={name} oninput={() => (nameTouched = true)} autocomplete="off" spellcheck="false" disabled={running} /></label>
  <div class="final selectable">{finalPath}</div>
  {#if url.trim() && sug && !sug.ok}
    <div class="hint err">{t('dlg.badUrl')}</div>
  {:else if problem}
    <div class="hint err">{t(problem.startsWith('name.') ? problem : 'name.' + problem)}</div>
  {:else if missingDirs && sug?.ok}
    <div class="hint">{t('dlg.willCreateDirs')}</div>
  {/if}
  {#if def}<label class="chk"><input type="checkbox" bind:checked={openAfter} disabled={running} /> {t('dlg.openAfter')}</label>{/if}
  {#if progress}
    <div class="prog"><i style="width:{progress.percent}%"></i></div>
    <div class="hint">{progress.phase ? `${phaseLabel(progress.phase)}: ${progress.percent}%` : t('dlg.cloning')}</div>
  {/if}
</div>
<div class="ft">
  <button onclick={cancel}>{t('dlg.cancel')}</button>
  <button class="p" disabled={!ok} onclick={run}>{running ? t('dlg.cloning') : t('dlg.cloneBtn')}</button>
</div>
