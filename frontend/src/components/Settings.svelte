<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { lib, errMessage, serviceName, type Launcher, type InstallInfo, type Preset } from '../lib/api'
  import { suggestedLauncher } from '../lib/presets'
  import LauncherIcon from './LauncherIcon.svelte'

  const sections = ['general', 'launchers', 'scan', 'git', 'exceptions', 'shortcuts', 'about'] as const
  let pb: HTMLDivElement | undefined = $state()

  function section(s: string) {
    store.settingsSection = s
    editing = null
    if (pb) pb.scrollTop = 0
  }

  // ---------- general ----------
  let rootExists = $state<Record<string, boolean>>({})
  $effect(() => {
    for (const r of store.cfg.roots) lib.Exists(r).then((ok) => (rootExists[r] = ok))
  })
  async function addRoot() {
    const p = await lib.PickFolder(t('settings.general.addRoot'), store.st?.home ?? '').catch(() => '')
    if (p && !store.cfg.roots.includes(p)) store.save((c) => c.roots.push(p))
  }
  let hotkey = $state(store.cfg.hotkey)
  let spotKey = $state(store.cfg.spotlightHotkey)

  // ---------- launcher ----------
  let editing = $state<string | null>(null)
  let draft = $state({ name: '', command: '', args: '' })

  function edit(l: Launcher) {
    if (editing === l.id) { editing = null; return }
    editing = l.id
    draft = { name: l.name, command: l.command, args: l.args }
  }
  function saveLauncher(id: string) {
    store.save((c) => {
      const l = c.launchers.find((x) => x.id === id)
      if (l) Object.assign(l, { name: draft.name.trim() || l.name, command: draft.command.trim(), args: draft.args })
    }).then((ok) => ok && store.toast(t('settings.launchers.saved')))
    editing = null
  }
  async function test(l: Launcher) {
    const target = store.target
    if (!target) return store.toast(t('settings.launchers.testNeedsProject'), true)
    try {
      await lib.TestLauncher({ ...$state.snapshot(l), name: draft.name, command: draft.command.trim(), args: draft.args }, target.path)
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }
  function move(i: number, d: number) {
    store.save((c) => {
      const [x] = c.launchers.splice(i, 1)
      c.launchers.splice(i + d, 0, x)
    })
  }
  function addLauncher() {
    const id = 'custom-' + Date.now()
    store.save((c) => c.launchers.push({ id, name: t('settings.launchers.newName'), command: '', args: '"{path}"', enabled: false })).then(() => {
      editing = id
      draft = { name: t('settings.launchers.newName'), command: '', args: '"{path}"' }
    })
  }
  // deletes a custom launcher together with the rules and per-project choices that use it
  function deleteLauncher(i: number) {
    editing = null
    store.save((c) => {
      const [gone] = c.launchers.splice(i, 1)
      c.rules = c.rules.filter((r) => r.launcher !== gone.id)
      for (const [p, id] of Object.entries(c.projectLaunchers)) if (id === gone.id) delete c.projectLaunchers[p]
    })
  }

  // ---------- rules (preset → launcher, in priority order) ----------
  let picking = $state(false)
  let allPresets = $derived([...store.catalog, ...store.cfg.presets])
  let usedPresets = $derived(new Set(store.cfg.rules.map((r) => r.preset)))

  function addRule(preset: Preset) {
    picking = false
    const id = suggestedLauncher(preset.id)?.id ?? ''
    store.save((c) => {
      c.rules.push({ preset: preset.id, launcher: id })
      // an installed but disabled editor gets enabled: otherwise the rule would be skipped
      const l = c.launchers.find((x) => x.id === id)
      if (l) l.enabled = true
    })
  }
  function moveRule(i: number, d: number) {
    store.save((c) => {
      const [r] = c.rules.splice(i, 1)
      c.rules.splice(i + d, 0, r)
    })
  }
  // "Customize": editable copy of the built-in preset, which takes its place in the rule
  function customize(i: number, p: Preset) {
    const id = 'custom-' + Date.now()
    store.save((c) => {
      c.presets.push({ id, name: t('settings.launchers.copyName', { name: p.name }), patterns: [...p.patterns], builtin: false })
      c.rules[i].preset = id
    })
  }

  // ---------- user presets ----------
  function newPreset() {
    picking = false
    const id = 'custom-' + Date.now()
    store.save((c) => c.presets.push({ id, name: t('settings.launchers.newPreset'), patterns: [], builtin: false }))
  }
  function presetIndex(c: { presets: Preset[] }, id: string) {
    return c.presets.findIndex((p) => p.id === id)
  }
  function renamePreset(id: string, name: string) {
    if (name.trim()) store.save((c) => (c.presets[presetIndex(c, id)].name = name.trim()))
  }
  function addPattern(e: KeyboardEvent, id: string) {
    const el = e.currentTarget as HTMLInputElement
    if (e.key !== 'Enter' || !el.value.trim()) return
    const v = el.value.trim()
    el.value = ''
    store.save((c) => {
      const p = c.presets[presetIndex(c, id)]
      if (!p.patterns.includes(v)) p.patterns.push(v)
    })
  }
  // deletes the preset and the rules that use it
  function deletePreset(id: string) {
    store.save((c) => {
      c.presets.splice(presetIndex(c, id), 1)
      c.rules = c.rules.filter((r) => r.preset !== id)
    })
  }

  let forcedProjects = $derived(Object.entries(store.cfg.projectLaunchers).sort(([a], [b]) => a.localeCompare(b)))

  async function detect() {
    const added = (await lib.SyncEditors()) ?? []
    const found: string[] = []
    for (const l of store.cfg.launchers.filter((x) => x.builtin)) if (await lib.DetectEditor(l.builtin!)) found.push(l.name)
    await store.refreshLaunchers()
    if (added.length) store.toast(t('settings.launchers.added', { list: added.join(', ') }))
    else store.toast(found.length ? t('settings.launchers.detected', { list: found.join(', ') }) : t('settings.launchers.detectedNone'))
  }

  // opened from the wizard: show the rules directly
  $effect(() => {
    const a = store.settingsAnchor
    if (!a || !pb) return
    if (a === 'rules' && !store.cfg.rules.length) picking = true
    queueMicrotask(() => document.getElementById(a)?.scrollIntoView({ block: 'start' }))
    store.settingsAnchor = ''
  })

  // ---------- export / import ----------
  async function exportConfig() {
    try {
      const p = await lib.ExportConfig(t('settings.about.export'))
      if (p) store.toast(t('settings.about.exported', { path: p }))
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }
  let importAsk = $state(false)
  async function importConfig() {
    importAsk = false
    try {
      const before = JSON.stringify(store.cfg)
      store.applyState(await lib.ImportConfig(t('settings.about.import')))
      if (JSON.stringify(store.cfg) === before) return // cancelled or unchanged
      store.setTree(await lib.Tree())
      store.refreshLaunchers()
      store.toast(t('settings.about.imported'))
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }
  // ---------- reset ----------
  let resetAsk = $state(false)
  let resetCancel: HTMLButtonElement | undefined = $state()
  $effect(() => { if (resetAsk) resetCancel?.focus() })
  async function resetConfig() {
    resetAsk = false
    try {
      store.applyState(await lib.ResetConfig())
      store.setTree(await lib.Tree())
      store.refreshLaunchers()
      store.go([])
      store.settingsOpen = false
      store.wizardOpen = true
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }
  async function browseExe() {
    const p = await lib.PickFile(t('settings.launchers.exe')).catch(() => '')
    if (p) draft.command = p
  }

  // ---------- scanning ----------
  function addChip(e: KeyboardEvent, key: 'markers' | 'ignore') {
    const el = e.currentTarget as HTMLInputElement
    if (e.key !== 'Enter' || !el.value.trim()) return
    const v = el.value.trim()
    el.value = ''
    if (!store.cfg[key].includes(v)) store.save((c) => c[key].push(v))
  }
  async function rescan() {
    store.setTree(await lib.Rescan())
    store.toast(t('settings.scan.rescanned', { n: store.tree?.count ?? 0 }))
  }

  // ---------- git ----------
  let gitPath = $state(store.cfg.gitPath)
  let install = $state<InstallInfo | null>(null)
  async function applyGit() {
    await store.save((c) => (c.gitPath = gitPath.trim()))
    if (!store.st?.gitAvailable) install = await lib.GitInstallInfo()
  }
  async function browseGit() {
    const p = await lib.PickFile(t('settings.git.path')).catch(() => '')
    if (p) { gitPath = p; applyGit() }
  }

  // ---------- GitHub and GitLab ----------
  let checking = $state(false)
  let cliInstall = $state<Record<string, InstallInfo>>({})
  let cliStarted = $state<Record<string, boolean>>({})
  let cliPaths = $state<Record<string, string>>({ github: store.cfg.ghPath, gitlab: store.cfg.glabPath })
  let copiedCmd = $state('')
  const binOf = (k: string) => (k === 'gitlab' ? 'glab' : 'gh')
  const configuredPath = (k: string) => (k === 'gitlab' ? store.cfg.glabPath : store.cfg.ghPath)
  // when the section opens: cached state, loaded at startup
  $effect(() => {
    if (store.settingsSection === 'git' && !store.forges.length) store.loadForges()
  })
  async function recheckForges() {
    checking = true
    await store.loadForges(true)
    checking = false
  }
  async function applyCliPath(k: string) {
    const p = (cliPaths[k] ?? '').trim()
    await store.save((c) => (k === 'gitlab' ? (c.glabPath = p) : (c.ghPath = p)))
    await recheckForges()
  }
  async function browseCli(k: string) {
    const p = await lib.PickFile(t('settings.git.cliPath')).catch(() => '')
    if (p) { cliPaths[k] = p; applyCliPath(k) }
  }
  async function installCli(k: string) {
    const info = await lib.ForgeInstallInfo(k)
    cliInstall[k] = info
    if (!info.canRun) return
    try {
      await lib.RunForgeInstall(k)
      cliStarted[k] = true
    } catch (e) {
      store.toast(errMessage(e), true)
    }
  }
  async function copyCmd(cmd: string) {
    await navigator.clipboard.writeText(cmd).catch(() => {})
    copiedCmd = cmd
    setTimeout(() => (copiedCmd = ''), 1500)
  }

  let overrides = $derived(Object.entries(store.cfg.overrides ?? {}).sort(([a], [b]) => a.localeCompare(b)))
  const configDir = $derived((store.st?.configPath ?? '').replace(/[\\/][^\\/]*$/, ''))
  // same file ResetConfig writes in the backend, next to config.json
  const backupPath = $derived((store.st?.configPath ?? '').replace(/[^\\/]*$/, 'config.backup.json'))

  function onkey(e: KeyboardEvent) {
    if (e.key !== 'Escape') return
    e.stopPropagation()
    if (resetAsk) resetAsk = false
    else store.settingsOpen = false
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="set-bg" onmousedown={(e) => e.target === e.currentTarget && (store.settingsOpen = false)} onkeydown={onkey}>
  <div class="set" role="dialog" aria-modal="true" aria-label={t('settings.title')}>
    <nav>
      <h2>{t('settings.title')}</h2>
      {#each sections as s}
        <button class:on={store.settingsSection === s} onclick={() => section(s)}>{t('settings.sections.' + s)}</button>
      {/each}
      <span class="sp"></span>
      <span class="ver">v{store.st?.version}</span>
    </nav>
    <div class="pane">
      <div class="ph"><h3>{t('settings.sections.' + store.settingsSection)}</h3><small>{t('settings.saved')}</small><button class="x" aria-label={t('settings.close')} onclick={() => (store.settingsOpen = false)}>✕</button></div>
      <div class="pb" bind:this={pb}>
        {#if store.settingsSection === 'general'}
          <div class="grp">{t('settings.general.roots')}</div>
          <div class="roots">
            {#each store.cfg.roots as r, i (r)}
              <div class="r"><span title={r}>{r}</span>{#if rootExists[r] === false}<span class="bad">{t('settings.general.missing')}</span>{/if}
                {#if store.cfg.roots.length > 1}<button class="btn d" onclick={() => store.save((c) => c.roots.splice(i, 1))}>{t('settings.general.remove')}</button>{/if}</div>
            {/each}
            <button class="btn" onclick={addRoot}>{t('settings.general.addRoot')}</button>
          </div>
          <div class="grp">{t('settings.general.appearance')}</div>
          <div class="f"><div class="l"><b>{t('settings.general.language')}</b></div>
            <select value={store.cfg.language} onchange={(e) => store.save((c) => (c.language = (e.currentTarget as HTMLSelectElement).value))}>
              <option value="en">English</option><option value="it">Italiano</option>
            </select></div>
          <div class="f"><div class="l"><b>{t('settings.general.theme')}</b><span>{t('settings.general.themeSub')}</span></div>
            <div class="seg">{#each ['system', 'light', 'dark'] as v}<button class:on={store.cfg.theme === v} onclick={() => store.save((c) => (c.theme = v))}>{t('settings.general.' + v)}</button>{/each}</div></div>
          <div class="grp">{t('settings.sections.general')}</div>
          <div class="f"><div class="l"><b>{t('settings.general.startMode')}</b><span>{t('settings.general.startModeSub')}</span></div>
            <div class="seg">{#each ['off', 'window', 'tray'] as v}<button class:on={store.cfg.startMode === v} onclick={() => store.save((c) => (c.startMode = v))}>{t('settings.general.start.' + v)}</button>{/each}</div></div>
          {#if store.cfg.startMode === 'tray' && !store.cfg.closeToTray}
            <div class="note warn">{t('settings.general.trayHint')} <button class="btn" onclick={() => store.save((c) => (c.closeToTray = true))}>{t('settings.general.trayHintBtn')}</button></div>
          {/if}
          <div class="f"><div class="l"><b>{t('settings.general.tray')}</b><span>{t('settings.general.traySub')}</span></div><input type="checkbox" class="sw" checked={store.cfg.closeToTray} onchange={(e) => store.save((c) => (c.closeToTray = (e.currentTarget as HTMLInputElement).checked))} /></div>
          <div class="grp">{t('settings.general.shortcutsGrp')}</div>
          <div class="f"><div class="l"><b>{t('settings.general.spotlightHotkey')}</b><span>{t('settings.general.spotlightHotkeySub')}</span></div>
            <input type="text" bind:value={spotKey} style="width:190px" onkeydown={(e) => e.key === 'Enter' && store.save((c) => (c.spotlightHotkey = spotKey.trim()))} />
            <button class="btn" disabled={spotKey.trim() === store.cfg.spotlightHotkey} onclick={() => store.save((c) => (c.spotlightHotkey = spotKey.trim()))}>{t('settings.general.apply')}</button></div>
          <div class="f"><div class="l"><b>{t('settings.general.hotkey')}</b><span>{t('settings.general.hotkeySub')}</span></div>
            <input type="text" bind:value={hotkey} style="width:190px" onkeydown={(e) => e.key === 'Enter' && store.save((c) => (c.hotkey = hotkey.trim()))} />
            <button class="btn" disabled={hotkey.trim() === store.cfg.hotkey} onclick={() => store.save((c) => (c.hotkey = hotkey.trim()))}>{t('settings.general.apply')}</button></div>
          <div class="note">{t('settings.general.hotkeyFormat')}</div>

        {:else if store.settingsSection === 'launchers'}
          <div class="grp">{t('settings.launchers.group')}</div>
          {#each store.cfg.launchers as l, i (l.id)}
            {@const found = store.launcherStatus[l.id]}
            <div class="ln">
              <div class="top">
                <div class="ord">
                  <button aria-label={t('settings.launchers.up')} disabled={i === 0} onclick={() => move(i, -1)}>▲</button>
                  <button aria-label={t('settings.launchers.down')} disabled={i === store.cfg.launchers.length - 1} onclick={() => move(i, 1)}>▼</button>
                </div>
                <LauncherIcon launcher={l} />
                <div class="nm"><b>{l.name}</b> {#if !l.builtin}<span class="tg">{t('settings.launchers.custom')}</span>{/if}
                  {#if found}<code title={found}>{found} {l.args}</code>{:else}<code class="miss">{t('settings.launchers.notFound')}</code>{/if}</div>
                <label class="def"><input type="radio" name="ldef" checked={store.cfg.defaultLauncher === l.id} disabled={!l.enabled} onchange={() => store.save((c) => (c.defaultLauncher = l.id))} /> {t('settings.launchers.default')}</label>
                <button class="btn" onclick={() => edit(l)}>{editing === l.id ? t('settings.launchers.close') : t('settings.launchers.edit')}</button>
                <input type="checkbox" class="sw" aria-label={l.name} checked={l.enabled} onchange={(e) => store.save((c) => (c.launchers[i].enabled = (e.currentTarget as HTMLInputElement).checked))} />
              </div>
              {#if editing === l.id}
                <div class="edit">
                  <label for="le-n">{t('settings.launchers.name')}</label><input id="le-n" type="text" bind:value={draft.name} />
                  <label for="le-c">{t('settings.launchers.exe')}</label>
                  <div class="row2"><input id="le-c" type="text" bind:value={draft.command} placeholder={l.builtin ? found : ''} /><button class="btn" onclick={browseExe}>{t('settings.launchers.browse')}</button></div>
                  {#if l.builtin}<div class="ph2">{t('settings.launchers.autoDetect')}</div>{/if}
                  <label for="le-a">{t('settings.launchers.args')}</label><input id="le-a" type="text" bind:value={draft.args} />
                  <div class="ph2">{@html t('settings.launchers.placeholders', { list: '<code>{path}</code> <code>{name}</code> <code>{source}</code> <code>{group}</code>' })}</div>
                  <div class="acts2">
                    {#if !l.builtin}<button class="btn d" onclick={() => deleteLauncher(i)}>{t('settings.launchers.delete')}</button>{/if}
                    <button class="btn" onclick={() => test(l)}>{t('settings.launchers.test')}</button>
                    <button class="btn p" onclick={() => saveLauncher(l.id)}>{t('settings.launchers.save')}</button>
                  </div>
                </div>
              {/if}
            </div>
          {/each}
          <div style="display:flex;gap:8px;margin-top:4px"><button class="btn" onclick={addLauncher}>{t('settings.launchers.add')}</button><button class="btn" onclick={detect}>{t('settings.launchers.detect')}</button></div>
          <div class="note">{t('settings.launchers.note', { mod: store.mod })}</div>

          <div class="grp" id="rules">{t('settings.launchers.rules')}</div>
          <p class="sub2">{t('settings.launchers.rulesText')}</p>
          {#each store.cfg.rules as r, i (r.preset + ':' + i)}
            {@const p = store.presetById(r.preset)}
            <div class="rule">
              <div class="ord">
                <button aria-label={t('settings.launchers.up')} disabled={i === 0} onclick={() => moveRule(i, -1)}>▲</button>
                <button aria-label={t('settings.launchers.down')} disabled={i === store.cfg.rules.length - 1} onclick={() => moveRule(i, 1)}>▼</button>
              </div>
              <div class="rb">
                <div><b>{p?.name ?? r.preset}</b>
                  {#if !p}<span class="tg bad">{t('settings.launchers.presetMissing')}</span>{:else if p.builtin}<span class="tg">{t('settings.launchers.builtinTag')}</span>{:else}<span class="tg">{t('settings.launchers.custom')}</span>{/if}</div>
                {#if p}<code class="pats" title={p.patterns.join('  ')}>{p.patterns.join('  ') || t('settings.launchers.noPatterns')}</code>{/if}
              </div>
              {#if p?.builtin}<button class="btn" title={t('settings.launchers.customizeSub')} onclick={() => customize(i, p)}>{t('settings.launchers.customize')}</button>{/if}
              <span class="lbl">{t('settings.launchers.openWith')}</span>
              <select value={r.launcher} onchange={(e) => store.save((c) => (c.rules[i].launcher = (e.currentTarget as HTMLSelectElement).value))}>
                {#each store.cfg.launchers as l (l.id)}<option value={l.id}>{l.name}{l.enabled ? '' : ` (${t('settings.launchers.disabled')})`}</option>{/each}
              </select>
              <button class="x" aria-label={t('settings.launchers.removeRule')} title={t('settings.launchers.removeRule')} onclick={() => store.save((c) => c.rules.splice(i, 1))}>✕</button>
            </div>
          {:else}
            <div class="note">{t('settings.launchers.noRules')}</div>
          {/each}
          {#if picking}
            <div class="pick">
              <div class="pk-h">{t('settings.launchers.library')}<button class="x" aria-label={t('settings.launchers.close')} onclick={() => (picking = false)}>✕</button></div>
              {#each allPresets as p (p.id)}
                {@const sl = suggestedLauncher(p.id)}
                <button class="pk" disabled={usedPresets.has(p.id)} onclick={() => addRule(p)}>
                  <span class="pn"><b>{p.name}</b>{#if !p.builtin}<span class="tg">{t('settings.launchers.custom')}</span>{/if}<code>{p.patterns.join('  ')}</code></span>
                  {#if usedPresets.has(p.id)}<span class="lbl">{t('settings.launchers.inUse')}</span>
                  {:else if sl}<span class="lbl">→</span><LauncherIcon launcher={sl} /><span class="sl">{sl.name}</span>{/if}
                </button>
              {/each}
            </div>
          {:else}
            <div style="display:flex;gap:8px;margin-top:4px"><button class="btn" onclick={() => (picking = true)}>{t('settings.launchers.addRule')}</button><button class="btn" onclick={newPreset}>{t('settings.launchers.addPreset')}</button></div>
          {/if}

          <div class="grp">{t('settings.launchers.presets')}</div>
          <p class="sub2">{t('settings.launchers.presetsText')}</p>
          {#each store.cfg.presets as p (p.id)}
            <div class="rule preset">
              <div class="rb">
                <input type="text" class="pname" aria-label={t('settings.launchers.name')} value={p.name} onchange={(e) => renamePreset(p.id, (e.currentTarget as HTMLInputElement).value)} />
                <div class="chips">
                  {#each p.patterns as pat, j (pat)}<span>{pat}<button aria-label="×" onclick={() => store.save((c) => c.presets[presetIndex(c, p.id)].patterns.splice(j, 1))}>×</button></span>{/each}
                  <input type="text" placeholder={t('settings.scan.addChip')} onkeydown={(e) => addPattern(e, p.id)} />
                </div>
              </div>
              {#if !usedPresets.has(p.id)}<button class="btn" onclick={() => addRule(p)}>{t('settings.launchers.useInRule')}</button>{/if}
              <button class="x" aria-label={t('settings.launchers.deletePreset')} title={t('settings.launchers.deletePreset')} onclick={() => deletePreset(p.id)}>✕</button>
            </div>
          {:else}
            <div class="note">{t('settings.launchers.noPresets')}</div>
          {/each}
          <div class="note">{t('settings.launchers.patternsHelp')}</div>

          <div class="grp">{t('settings.launchers.forced')}</div>
          {#if forcedProjects.length}
            <table class="tbl"><tbody>
              {#each forcedProjects as [path, id] (path)}
                {@const l = store.cfg.launchers.find((x) => x.id === id)}
                <tr><td class="mono">{path}</td><td>{l?.name ?? id}</td>
                  <td><button class="btn" onclick={() => store.save((c) => delete c.projectLaunchers[path])}>{t('settings.launchers.remove')}</button></td></tr>
              {/each}
            </tbody></table>
          {:else}
            <div class="note">{t('settings.launchers.forcedNone')}</div>
          {/if}

        {:else if store.settingsSection === 'scan'}
          <div class="grp">{t('settings.scan.detection')}</div>
          <div class="f"><div class="l"><b>{t('settings.scan.markers')}</b><span>{t('settings.scan.markersSub')}</span></div></div>
          <div class="chips">
            {#each store.cfg.markers as m, i (m)}<span>{m}<button aria-label="×" onclick={() => store.save((c) => c.markers.splice(i, 1))}>×</button></span>{/each}
            <input type="text" placeholder={t('settings.scan.addChip')} onkeydown={(e) => addChip(e, 'markers')} />
          </div>
          <div class="f"><div class="l"><b>{t('settings.scan.files')}</b><span>{t('settings.scan.filesSub')}</span></div><input type="checkbox" class="sw" checked={store.cfg.filesAsProject} onchange={(e) => store.save((c) => (c.filesAsProject = (e.currentTarget as HTMLInputElement).checked))} /></div>
          <div class="f"><div class="l"><b>{t('settings.scan.empty')}</b><span>{t('settings.scan.emptySub')}</span></div><input type="checkbox" class="sw" checked={store.cfg.showEmpty} onchange={(e) => store.save((c) => (c.showEmpty = (e.currentTarget as HTMLInputElement).checked))} /></div>
          <div class="grp">{t('settings.scan.exclusions')}</div>
          <div class="f"><div class="l"><b>{t('settings.scan.ignore')}</b><span>{t('settings.scan.ignoreSub')}</span></div></div>
          <div class="chips">
            {#each store.cfg.ignore as m, i (m)}<span>{m}<button aria-label="×" onclick={() => store.save((c) => c.ignore.splice(i, 1))}>×</button></span>{/each}
            <input type="text" placeholder={t('settings.scan.addChip')} onkeydown={(e) => addChip(e, 'ignore')} />
          </div>
          <div class="grp">{t('settings.scan.advanced')}</div>
          <div class="f"><div class="l"><b>{t('settings.scan.depth')}</b><span>{t('settings.scan.depthSub')}</span></div><input type="number" min="1" max="64" style="width:80px" value={store.cfg.maxDepth} onchange={(e) => store.save((c) => (c.maxDepth = +(e.currentTarget as HTMLInputElement).value))} /></div>
          <div class="f"><div class="l"><b>{t('settings.scan.links')}</b><span>{t('settings.scan.linksSub')}</span></div><input type="checkbox" class="sw" checked={store.cfg.followLinks} onchange={(e) => store.save((c) => (c.followLinks = (e.currentTarget as HTMLInputElement).checked))} /></div>
          <div class="f"><div class="l"><b>{t('settings.scan.rescan')}</b><span>{t('settings.scan.rescanSub', { n: store.tree?.count ?? 0 })}</span></div><button class="btn" onclick={rescan}>{t('settings.scan.rescanNow')}</button></div>

        {:else if store.settingsSection === 'git'}
          <div class="grp">{t('settings.git.card')}</div>
          <div class="f"><div class="l"><b>{t('settings.git.info')}</b><span>{t('settings.git.infoSub')}</span></div><input type="checkbox" class="sw" checked={store.cfg.gitInfo} onchange={(e) => store.save((c) => (c.gitInfo = (e.currentTarget as HTMLInputElement).checked))} /></div>
          <div class="f"><div class="l"><b>{t('settings.git.fetch')}</b><span>{t('settings.git.fetchSub')}</span></div><input type="checkbox" class="sw" checked={store.cfg.gitFetch} onchange={(e) => store.save((c) => (c.gitFetch = (e.currentTarget as HTMLInputElement).checked))} /></div>
          <div class="f"><div class="l"><b>{t('settings.git.interval')}</b><span>{t('settings.git.intervalSub')}</span></div><input type="number" min="5" style="width:80px" disabled={!store.cfg.gitFetch} value={store.cfg.gitFetchMinutes} onchange={(e) => store.save((c) => (c.gitFetchMinutes = +(e.currentTarget as HTMLInputElement).value))} /></div>
          <div class="grp">{t('settings.git.exe')}</div>
          <div class="f"><div class="l"><b>{t('settings.git.path')}</b><span>{store.st?.gitAvailable ? t('settings.git.found', { path: store.st.gitPath }) : t('settings.git.notFound')}</span></div>
            <input type="text" bind:value={gitPath} placeholder={t('settings.git.pathSub')} style="width:260px" onkeydown={(e) => e.key === 'Enter' && applyGit()} />
            <button class="btn" onclick={browseGit}>{t('settings.launchers.browse')}</button>
            <button class="btn" onclick={applyGit}>{t('settings.git.detect')}</button></div>
          {#if !store.st?.gitAvailable}
            {#if install}
              <div class="note warn">{install.command ? install.command + ' · ' : ''}<a href={install.url} onclick={(e) => { e.preventDefault(); lib.OpenURL(install!.url) }}>{install.url}</a></div>
            {:else}
              <button class="btn" style="margin-top:10px" onclick={async () => (install = await lib.GitInstallInfo())}>{t('settings.git.install')}</button>
            {/if}
          {/if}

          <div class="grp">{t('settings.git.forges')}</div>
          <p class="sub2">{t('settings.git.forgesText')}</p>
          {#each store.forges as f (f.kind)}
            {@const k = String(f.kind)}
            {@const bin = binOf(k)}
            <div class="cli">
              <div class="f"><div class="l"><b>{serviceName(k)} CLI <code>{bin}</code></b>
                <span title={f.path}>{f.path ? t('settings.git.cliFound', { path: f.path }) : t('settings.git.cliNotFound')}</span></div>
                {#if !f.path}<button class="btn" onclick={() => installCli(k)}>{t('settings.git.installCli')}</button>{/if}
              </div>
              {#if f.path}
                {#each f.accounts ?? [] as a (a.host)}
                  <div class="acc">✓ {t('settings.git.loggedAs', { user: a.user, host: a.host })} <span class="proto">{a.protocol}</span></div>
                {:else}
                  <div class="note warn">{t('settings.git.notLogged')} <code>{bin} auth login</code>
                    <button class="btn" onclick={() => copyCmd(bin + ' auth login')}>{copiedCmd === bin + ' auth login' ? t('settings.git.copied') : t('settings.git.copy')}</button></div>
                {/each}
              {:else if cliInstall[k]}
                {@const inf = cliInstall[k]}
                {#if cliStarted[k]}
                  <div class="note">{t('settings.git.installStarted')}</div>
                {:else}
                  <div class="note warn">
                    {#if inf.command}{t('settings.git.installManual')} <code>{inf.command}</code>
                      <button class="btn" onclick={() => copyCmd(inf.command)}>{copiedCmd === inf.command ? t('settings.git.copied') : t('settings.git.copy')}</button> · {/if}
                    <a href={inf.url} onclick={(e) => { e.preventDefault(); lib.OpenURL(inf.url) }}>{inf.url}</a>
                  </div>
                {/if}
              {/if}
              {#if !f.path || configuredPath(k)}
                <div class="f"><div class="l"><b>{t('settings.git.cliPath')}</b><span>{t('settings.git.cliPathSub')}</span></div>
                  <input type="text" bind:value={cliPaths[k]} style="width:260px" onkeydown={(e) => e.key === 'Enter' && applyCliPath(k)} />
                  <button class="btn" onclick={() => browseCli(k)}>{t('settings.launchers.browse')}</button></div>
              {/if}
            </div>
          {/each}
          <button class="btn" style="margin-top:12px" disabled={checking} onclick={recheckForges}>{checking ? t('settings.git.checking') : t('settings.git.recheck')}</button>

        {:else if store.settingsSection === 'exceptions'}
          <div class="grp">{t('settings.exceptions.group')}</div>
          <p style="color:var(--mute);margin:4px 0 10px">{t('settings.exceptions.text')}</p>
          {#if overrides.length}
            <table class="tbl"><tbody>
              {#each overrides as [path, kind] (path)}
                <tr><td class="mono">{path}</td><td>{kind === 'project' ? t('settings.exceptions.project') : t('settings.exceptions.dir')}</td>
                  <td><button class="btn" onclick={() => store.setOverride(path, '')}>{t('settings.exceptions.remove')}</button></td></tr>
              {/each}
            </tbody></table>
          {:else}
            <div class="note">{t('settings.exceptions.none')}</div>
          {/if}

        {:else if store.settingsSection === 'shortcuts'}
          <div class="grp">{t('settings.shortcuts.group')}</div>
          <table class="tbl"><tbody>
            {#each [
              [store.mod + ' K', 'search'], ['>', 'commands'], [store.mod + ' P  /  ' + store.mod + ' ,', 'settings'], ['↵', 'open'], [store.mod + ' 1…9', 'openN'],
              [store.mod + ' E', 'reveal'], [store.mod + ' Shift N', 'newFolder'], ['F2', 'rename'], [store.os === 'darwin' ? '⌘ ⌫' : 'Del', 'delete'],
              ['← ↑ ↓ →', 'navigate'], ['Esc', 'close'],
            ] as [k, d]}
              <tr><td>{t('settings.shortcuts.' + d)}</td><td><kbd>{k}</kbd></td></tr>
            {/each}
          </tbody></table>
          <div class="grp">{t('settings.shortcuts.globalGrp')}</div>
          <table class="tbl"><tbody>
            <tr><td>{t('settings.shortcuts.spotlight')}</td><td>{#if store.cfg.spotlightHotkey}<kbd>{store.cfg.spotlightHotkey}</kbd>{:else}{t('settings.shortcuts.off')}{/if}</td></tr>
            <tr><td>{t('settings.shortcuts.showMain')}</td><td>{#if store.cfg.hotkey}<kbd>{store.cfg.hotkey}</kbd>{:else}{t('settings.shortcuts.off')}{/if}</td></tr>
          </tbody></table>
          <div class="note">{t('settings.shortcuts.globalNote')}</div>

        {:else if store.settingsSection === 'about'}
          <div class="grp">Project Library</div>
          <div class="f"><div class="l"><b>{t('settings.about.version')}</b><span>{store.st?.version} · {store.os}</span></div></div>
          <div class="f"><div class="l"><b>{t('settings.about.config')}</b><span class="selectable">{store.st?.configPath}</span></div><button class="btn" onclick={() => store.reveal(configDir)}>{t('settings.about.openFolder')}</button></div>
          <div class="f"><div class="l"><b>{t('settings.about.export')}</b><span>{t('settings.about.exportSub')}</span></div><button class="btn" onclick={exportConfig}>{t('settings.about.exportBtn')}</button></div>
          <div class="f"><div class="l"><b>{t('settings.about.import')}</b><span>{t('settings.about.importSub')}</span></div><button class="btn" onclick={() => (importAsk = !importAsk)}>{t('settings.about.importBtn')}</button></div>
          {#if importAsk}
            <div class="note warn">{t('settings.about.importConfirm')} <button class="btn d" onclick={importConfig}>{t('settings.about.importGo')}</button> <button class="btn" onclick={() => (importAsk = false)}>{t('settings.about.cancel')}</button></div>
          {/if}
          <div class="f"><div class="l"><b>{t('settings.about.reset')}</b><span>{t('settings.about.resetSub')}</span></div><button class="btn danger" onclick={() => (resetAsk = true)}>{t('settings.about.resetBtn')}</button></div>
          {#if resetAsk}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="backdrop over" onmousedown={(e) => e.target === e.currentTarget && (resetAsk = false)}>
              <div class="dlg" role="alertdialog" aria-modal="true" aria-labelledby="reset-title">
                <h3 id="reset-title">{t('settings.about.resetTitle')}</h3>
                <div class="bd">
                  <p>{t('settings.about.resetText')}</p>
                  <p class="hint">{t('settings.about.resetBackup')}</p>
                  <div class="final selectable">{backupPath}</div>
                </div>
                <div class="ft">
                  <button bind:this={resetCancel} onclick={() => (resetAsk = false)}>{t('settings.about.cancel')}</button>
                  <button class="d" onclick={resetConfig}>{t('settings.about.resetGo')}</button>
                </div>
              </div>
            </div>
          {/if}
        {/if}
      </div>
    </div>
  </div>
</div>
