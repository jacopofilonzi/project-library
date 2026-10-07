<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t, setLang } from '../lib/i18n/index.svelte'
  import { lib, errMessage, type InstallInfo, type Launcher } from '../lib/api'
  import LauncherIcon from './LauncherIcon.svelte'

  // ---------- bozza della configurazione ----------
  let lang = $state(store.cfg.language)
  let roots = $state<string[]>([...store.cfg.roots])
  let exists = $state<Record<string, boolean>>({})
  type EdDraft = { launcher: Launcher; enabled: boolean; command: string; detected: string }
  let editors = $state<EdDraft[]>(
    store.cfg.launchers.map((l) => ({ launcher: $state.snapshot(l) as Launcher, enabled: l.enabled, command: l.command, detected: '' })),
  )
  let customs = $state<{ name: string; command: string }[]>([])
  let gitPath = $state(store.cfg.gitPath)
  let gitFound = $state(store.st?.gitAvailable ? store.st.gitPath : '')
  const needGit = !store.st?.gitAvailable
  let install = $state<InstallInfo | null>(null)
  let installStarted = $state(false)
  let gitChecked = $state(false)
  let copied = $state(false)

  const steps = needGit ? ['lang', 'roots', 'editors', 'git'] : ['lang', 'roots', 'editors']
  let step = $state(0)
  let busy = $state(false)

  // rilevamento iniziale di editor e git
  $effect(() => {
    for (const ed of editors) if (ed.launcher.builtin) lib.DetectEditor(ed.launcher.builtin).then((p) => (ed.detected = p))
    if (needGit) lib.GitInstallInfo().then((i) => (install = i))
  })
  $effect(() => {
    for (const r of roots) lib.Exists(r).then((ok) => (exists[r] = ok))
  })

  function chooseLang(l: string) {
    lang = l
    setLang(l)
  }

  async function addRoot() {
    const p = await lib.PickFolder(t('wizard.roots.add'), store.st?.home ?? '').catch(() => '')
    if (p && !roots.includes(p)) roots.push(p)
  }
  async function createRoot(r: string) {
    try {
      await lib.CreateRoot(r)
      exists[r] = true
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }
  async function browse(ed: EdDraft) {
    const p = await lib.PickFile(ed.launcher.name).catch(() => '')
    if (p) ed.command = p
  }
  async function browseCustom(i: number) {
    const p = await lib.PickFile(t('wizard.editors.customExe')).catch(() => '')
    if (p) customs[i].command = p
  }

  async function runInstall() {
    try {
      await lib.RunGitInstall()
      installStarted = true
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }
  async function recheck() {
    gitFound = await lib.DetectGit(gitPath.trim())
    gitChecked = true
  }
  async function copy(text: string) {
    await navigator.clipboard.writeText(text)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }

  let canNext = $derived(steps[step] !== 'roots' || roots.length > 0)

  async function finish() {
    busy = true
    const ok = await store.save((c) => {
      c.language = lang
      c.roots = [...roots]
      c.launchers = editors.map((e) => ({ ...e.launcher, enabled: e.enabled, command: e.command.trim() }))
      for (const [i, cu] of customs.entries()) {
        if (cu.name.trim() && cu.command.trim())
          c.launchers.push({ id: `custom-${Date.now()}-${i}`, name: cu.name.trim(), command: cu.command.trim(), args: '"{path}"', enabled: true })
      }
      if (!c.launchers.some((l) => l.id === c.defaultLauncher && l.enabled)) c.defaultLauncher = c.launchers.find((l) => l.enabled)?.id ?? ''
      c.gitPath = gitPath.trim()
      c.gitWarningShown = true
      c.setupDone = true
    })
    if (ok) {
      store.setTree(await lib.Rescan())
      store.go([])
      store.wizardOpen = false
    }
    busy = false
  }
</script>

<div class="wiz-bg">
  <div class="wiz" role="dialog" aria-modal="true">
    <div class="steps">{#each steps as _, i}<i class:on={i <= step}></i>{/each}</div>
    <div class="bd">
      {#if steps[step] === 'lang'}
        <h2>{t('wizard.lang.title')}</h2>
        <p class="sub">{t('wizard.lang.sub')}</p>
        {#each [['en', 'English'], ['it', 'Italiano']] as [code, label]}
          <button class="opt" class:on={lang === code} onclick={() => chooseLang(code)}><b>{label}</b>{#if lang === code}✓{/if}</button>
        {/each}

      {:else if steps[step] === 'roots'}
        <h2>{t('wizard.roots.title')}</h2>
        <p class="sub">{t('wizard.roots.sub')}</p>
        {#each roots as r, i (r)}
          <div class="ed">
            <div class="m"><b class="selectable">{r}</b>{#if exists[r] === false}<code>{t('wizard.roots.missing')}</code>{/if}</div>
            {#if exists[r] === false}<button class="btn" onclick={() => createRoot(r)}>{t('wizard.roots.create')}</button>{/if}
            <button class="btn d" onclick={() => roots.splice(i, 1)}>{t('wizard.roots.remove')}</button>
          </div>
        {/each}
        {#if !roots.length}<div class="note warn">{t('wizard.roots.needOne')}</div>{/if}
        <button class="btn" style="align-self:flex-start" onclick={addRoot}>{t('wizard.roots.add')}</button>

      {:else if steps[step] === 'editors'}
        <h2>{t('wizard.editors.title')}</h2>
        <p class="sub">{t('wizard.editors.sub')}</p>
        {#each editors as ed (ed.launcher.id)}
          {@const found = ed.command.trim() || ed.detected}
          <div class="ed">
            <LauncherIcon launcher={ed.launcher} />
            <div class="m">
              <b>{ed.launcher.name}</b>
              {#if found}<span class="ok"> · {t('wizard.editors.found')}</span>{:else}<span class="ko"> · {t('wizard.editors.notFound')}</span>{/if}
              <div style="display:flex;gap:6px;margin-top:6px">
                <input type="text" style="flex:1;min-width:0" bind:value={ed.command} placeholder={ed.detected || t('wizard.editors.path')} spellcheck="false" />
                <button class="btn" onclick={() => browse(ed)}>{t('wizard.editors.browse')}</button>
              </div>
            </div>
            <input type="checkbox" class="sw" aria-label={ed.launcher.name} bind:checked={ed.enabled} />
          </div>
        {/each}
        <p class="sub">{t('wizard.editors.custom')}</p>
        {#each customs as cu, i}
          <div class="ed">
            <div class="m" style="display:flex;gap:6px">
              <input type="text" style="width:150px" bind:value={cu.name} placeholder={t('wizard.editors.customName')} />
              <input type="text" style="flex:1;min-width:0" bind:value={cu.command} placeholder={t('wizard.editors.customExe')} spellcheck="false" />
              <button class="btn" onclick={() => browseCustom(i)}>{t('wizard.editors.browse')}</button>
            </div>
            <button class="btn d" onclick={() => customs.splice(i, 1)}>✕</button>
          </div>
        {/each}
        <button class="btn" style="align-self:flex-start" onclick={() => customs.push({ name: '', command: '' })}>{t('wizard.editors.addCustom')}</button>

      {:else if steps[step] === 'git'}
        <h2>{gitFound ? t('wizard.git.found', { path: gitFound }) : t('wizard.git.title')}</h2>
        {#if !gitFound}
          <p class="sub">{t('wizard.git.sub')}</p>
          {#if install?.canRun}
            <button class="btn p" style="align-self:flex-start" onclick={runInstall}>{t('wizard.git.install')}</button>
            {#if installStarted}<div class="note">{t('wizard.git.installing')}</div>{/if}
          {/if}
          {#if install?.command}
            <p class="sub">{t('wizard.git.manual')}</p>
            <code class="cmd"><span class="selectable">{install.command}</span><button class="btn" onclick={() => copy(install!.command)}>{copied ? t('wizard.git.copied') : t('wizard.git.copy')}</button></code>
          {/if}
          {#if install?.url}<button class="btn" style="align-self:flex-start" onclick={() => lib.OpenURL(install!.url)}>{t('wizard.git.download')} ↗</button>{/if}
          <p class="sub">{t('wizard.git.path')}</p>
          <div style="display:flex;gap:6px"><input type="text" style="flex:1" bind:value={gitPath} spellcheck="false" /><button class="btn" onclick={recheck}>{t('wizard.git.recheck')}</button></div>
          {#if gitChecked && !gitFound}<div class="note warn">{t('wizard.git.stillMissing')}</div>{/if}
        {/if}
      {/if}
    </div>
    <div class="ft">
      {#if step > 0}<button class="btn" onclick={() => step--}>{t('wizard.back')}</button>{/if}
      <span class="sp"></span>
      {#if steps[step] === 'git' && !gitFound}<button class="btn" disabled={busy} onclick={finish}>{t('wizard.skip')}</button>{/if}
      {#if step < steps.length - 1}
        <button class="btn p" disabled={!canNext} onclick={() => step++}>{t('wizard.next')}</button>
      {:else}
        <button class="btn p" disabled={busy || !roots.length || (steps[step] === 'git' && !gitFound)} onclick={finish}>{t('wizard.finish')}</button>
      {/if}
    </div>
  </div>
</div>
