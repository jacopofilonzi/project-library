<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { lib, errMessage, type Launcher, type InstallInfo } from '../lib/api'
  import LauncherIcon from './LauncherIcon.svelte'

  const sections = ['general', 'launchers', 'scan', 'git', 'exceptions', 'shortcuts', 'about'] as const
  let pb: HTMLDivElement | undefined = $state()

  function section(s: string) {
    store.settingsSection = s
    editing = null
    if (pb) pb.scrollTop = 0
  }

  // ---------- generale ----------
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
  // elimina un launcher custom insieme alle regole e alle scelte per progetto che lo usano
  function deleteLauncher(i: number) {
    editing = null
    store.save((c) => {
      const [gone] = c.launchers.splice(i, 1)
      c.launcherRules = c.launcherRules.filter((r) => r.launcher !== gone.id)
      for (const [p, id] of Object.entries(c.projectLaunchers)) if (id === gone.id) delete c.projectLaunchers[p]
    })
  }

  // ---------- regole ----------
  function addRule() {
    const id = store.cfg.launchers.find((l) => l.builtin === 'intellij')?.id ?? store.cfg.launchers[0]?.id ?? ''
    store.save((c) => c.launcherRules.push({ patterns: [], launcher: id }))
  }
  function moveRule(i: number, d: number) {
    store.save((c) => {
      const [r] = c.launcherRules.splice(i, 1)
      c.launcherRules.splice(i + d, 0, r)
    })
  }
  function addPattern(e: KeyboardEvent, i: number) {
    const el = e.currentTarget as HTMLInputElement
    if (e.key !== 'Enter' || !el.value.trim()) return
    const v = el.value.trim()
    el.value = ''
    if (!store.cfg.launcherRules[i].patterns.includes(v)) store.save((c) => c.launcherRules[i].patterns.push(v))
  }
  let forcedProjects = $derived(Object.entries(store.cfg.projectLaunchers).sort(([a], [b]) => a.localeCompare(b)))

  async function detect() {
    const found: string[] = []
    for (const l of store.cfg.launchers.filter((x) => x.builtin)) if (await lib.DetectEditor(l.builtin!)) found.push(l.name)
    await store.refreshLaunchers()
    store.toast(found.length ? t('settings.launchers.detected', { list: found.join(', ') }) : t('settings.launchers.detectedNone'))
  }
  async function browseExe() {
    const p = await lib.PickFile(t('settings.launchers.exe')).catch(() => '')
    if (p) draft.command = p
  }

  // ---------- scansione ----------
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

  let overrides = $derived(Object.entries(store.cfg.overrides ?? {}).sort(([a], [b]) => a.localeCompare(b)))
  const configDir = $derived((store.st?.configPath ?? '').replace(/[\\/][^\\/]*$/, ''))

  function onkey(e: KeyboardEvent) {
    if (e.key === 'Escape') { e.stopPropagation(); store.settingsOpen = false }
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

          <div class="grp">{t('settings.launchers.rules')}</div>
          <p class="sub2">{t('settings.launchers.rulesText')}</p>
          {#each store.cfg.launcherRules as r, i (i)}
            <div class="rule">
              <div class="ord">
                <button aria-label={t('settings.launchers.up')} disabled={i === 0} onclick={() => moveRule(i, -1)}>▲</button>
                <button aria-label={t('settings.launchers.down')} disabled={i === store.cfg.launcherRules.length - 1} onclick={() => moveRule(i, 1)}>▼</button>
              </div>
              <div class="rb">
                <span class="lbl">{t('settings.launchers.ifContains')}</span>
                <div class="chips">
                  {#each r.patterns as p, j (p)}<span>{p}<button aria-label="×" onclick={() => store.save((c) => c.launcherRules[i].patterns.splice(j, 1))}>×</button></span>{/each}
                  <input type="text" placeholder={t('settings.scan.addChip')} onkeydown={(e) => addPattern(e, i)} />
                </div>
              </div>
              <span class="lbl">{t('settings.launchers.openWith')}</span>
              <select value={r.launcher} onchange={(e) => store.save((c) => (c.launcherRules[i].launcher = (e.currentTarget as HTMLSelectElement).value))}>
                {#each store.cfg.launchers as l (l.id)}<option value={l.id}>{l.name}{l.enabled ? '' : ` (${t('settings.launchers.disabled')})`}</option>{/each}
              </select>
              <button class="x" aria-label={t('settings.launchers.removeRule')} title={t('settings.launchers.removeRule')} onclick={() => store.save((c) => c.launcherRules.splice(i, 1))}>✕</button>
            </div>
          {:else}
            <div class="note">{t('settings.launchers.noRules')}</div>
          {/each}
          <button class="btn" style="margin-top:4px" onclick={addRule}>{t('settings.launchers.addRule')}</button>

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
              [store.mod + ' E', 'reveal'], [store.mod + ' Shift N', 'newFolder'], ['F2', 'rename'], [store.os === 'darwin' ? '⌘ ⌫' : 'Canc / Del', 'delete'],
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
          <div class="f"><div class="l"><b>{t('settings.about.wizard')}</b><span>{t('settings.about.wizardSub')}</span></div><button class="btn" onclick={() => { store.settingsOpen = false; store.wizardOpen = true }}>{t('settings.about.runWizard')}</button></div>
        {/if}
      </div>
    </div>
  </div>
</div>
