// Global app state (Svelte 5 runes).
import { Events } from '@wailsio/runtime'
import { lib, errMessage, normalizeState, normalizeConfig, type AppState, type Config, type Node, type Launcher, type Preset, type ForgeStatus, type ForgeAccount, type ForgeKind, type UpdateInfo } from './api'
import { setLang, t } from './i18n/index.svelte'

export type Dialog =
  | { kind: 'newFolder'; parent: string }
  | { kind: 'rename'; path: string; name: string }
  | { kind: 'delete'; node: Node }
  | { kind: 'clone'; parent: string | null }
  | { kind: 'init'; path: string; name: string }
  | { kind: 'publish'; node: Node }
  | { kind: 'move'; node: Node; dest?: string }

export type CtxItem = { label: string; key?: string; danger?: boolean; run: () => void } | '-'
export type Ctx = { x: number; y: number; items: CtxItem[] }

export type Toast = { id: number; text: string; err?: boolean }

export type Indexed = { node: Node; names: string[]; parent: string[] }

class Store {
  ready = $state(false)
  st = $state<AppState | null>(null)
  tree = $state<Node | null>(null)
  dirty = $state<Record<string, number>>({})
  launcherStatus = $state<Record<string, string>>({})
  /** catalog of the built-in presets (from the app) */
  catalog = $state<Preset[]>([])
  /** state of gh and glab (path and accounts), loaded in the background */
  forges = $state<ForgeStatus[]>([])
  /** result of the last update check; the banner hides when dismissed, until the next start */
  update = $state<UpdateInfo | null>(null)
  updateDismissed = $state(false)
  /** grows when an action changes the git state of the open project (the card reloads it) */
  gitChanged = $state(0)

  /** current path as names starting from the tree root */
  path = $state<string[]>([])
  /** name of the selected project in the current folder */
  sel = $state<string | null>(null)

  dialog = $state<Dialog | null>(null)
  ctx = $state<Ctx | null>(null)
  paletteOpen = $state(false)
  settingsOpen = $state(false)
  settingsSection = $state('general')
  wizardOpen = $state(false)
  toasts = $state<Toast[]>([])

  get cfg(): Config {
    return this.st!.config
  }
  get os() {
    return this.st?.os ?? 'windows'
  }
  get mod() {
    return this.os === 'darwin' ? '⌘' : 'Ctrl'
  }
  get enabledLaunchers(): Launcher[] {
    return (this.cfg.launchers ?? []).filter((l) => l.enabled)
  }
  get defaultLauncher(): Launcher | undefined {
    const ls = this.enabledLaunchers
    return ls.find((l) => l.id === this.cfg.defaultLauncher) ?? ls[0]
  }

  /**
   * Launcher to open node with, in order:
   * 1. chosen for that project (projectLaunchers), if enabled
   * 2. first rule that matches the project's files, with an enabled launcher
   * 3. default launcher
   */
  launcherFor(node: Node): { launcher: Launcher | undefined; reason: 'project' | 'rule' | 'default' } {
    const forced = this.enabledLaunchers.find((l) => l.id === this.cfg.projectLaunchers[node.path])
    if (forced) return { launcher: forced, reason: 'project' }
    return this.autoLauncherFor(node)
  }

  /** automatic choice (rules, then default), ignoring the manual one */
  autoLauncherFor(node: Node): { launcher: Launcher | undefined; reason: 'rule' | 'default' } {
    const ls = this.enabledLaunchers
    for (const id of node.ruleLaunchers ?? []) {
      const l = ls.find((x) => x.id === id)
      if (l) return { launcher: l, reason: 'rule' }
    }
    return { launcher: this.defaultLauncher, reason: 'default' }
  }

  /** preset by id, looked up in the catalog and then among the user's (as in the backend) */
  presetById(id: string): Preset | undefined {
    return this.catalog.find((p) => p.id === id) ?? this.cfg.presets.find((p) => p.id === id)
  }

  /** sets (id) or removes ('') the fixed launcher of a project */
  async setProjectLauncher(node: Node, id: string) {
    const ok = await this.save((c) => {
      if (id) c.projectLaunchers[node.path] = id
      else delete c.projectLaunchers[node.path]
    })
    if (!ok) return
    const l = this.cfg.launchers.find((x) => x.id === id)
    this.toast(l ? t('open.alwaysSaved', { name: node.name, launcher: l.name }) : t('open.autoSaved', { name: node.name }))
  }

  /** root label: the folder name, or "Roots" with several roots */
  get rootLabel(): string {
    if (!this.tree) return ''
    return this.tree.kind === 'root' ? t('roots') : this.tree.name
  }

  // ---------- tree ----------
  nodeAt(names: string[]): Node | null {
    let n = this.tree
    for (const name of names) {
      n = (n?.children ?? []).find((c) => c?.name === name && c.kind !== 'project') ?? null
      if (!n) return null
    }
    return n
  }
  get current(): Node | null {
    return this.nodeAt(this.path)
  }
  get selected(): Node | null {
    if (!this.sel) return null
    return (this.current?.children ?? []).find((c) => c?.kind === 'project' && c.name === this.sel) ?? null
  }
  /** what the launchers open: the selected project or the current empty folder */
  get target(): Node | null {
    if (this.selected) return this.selected
    const cur = this.current
    if (cur && this.path.length && cur.kind === 'empty' && !cur.missing) return cur
    return null
  }

  /** index of all projects: absolute path → node and names from the root */
  index = $derived.by(() => {
    const map = new Map<string, Indexed>()
    const walk = (n: Node, names: string[]) => {
      for (const c of n.children ?? []) {
        if (!c) continue
        if (c.kind === 'project') map.set(c.path, { node: c, names: [...names, c.name], parent: names })
        else walk(c, [...names, c.name])
      }
    }
    if (this.tree) walk(this.tree, [])
    return map
  })

  /** names from the root for an absolute folder path (null if it is not in the tree) */
  namesOf(path: string): string[] | null {
    let found: string[] | null = null
    const walk = (n: Node, names: string[]) => {
      if (found) return
      if (n.path === path) {
        found = names
        return
      }
      for (const c of n.children ?? []) if (c) walk(c, [...names, c.name])
    }
    if (this.tree) walk(this.tree, [])
    return found
  }

  /** after a new tree: keeps the longest valid path */
  private fixPath() {
    let p = [...this.path]
    while (p.length && !this.nodeAt(p)) p.pop()
    if (p.length !== this.path.length) {
      this.path = p
      this.sel = null
    }
    if (this.sel && !this.selected) this.sel = null
  }

  setTree(tree: Node | null) {
    this.tree = tree
    this.fixPath()
  }

  go(names: string[], sel: string | null = null) {
    this.path = names
    this.sel = sel
    this.rememberLocation()
  }

  // save where you are (with a small delay, keyboard navigation is fast)
  private locTimer: ReturnType<typeof setTimeout> | undefined
  private rememberLocation() {
    if (isSpotlight || !this.ready) return
    clearTimeout(this.locTimer)
    this.locTimer = setTimeout(() => {
      lib.SetLastLocation(this.current?.path ?? '', this.selected?.path ?? '').catch(() => {})
    }, 600)
  }

  // ---------- moving ----------
  /** path of the item being dragged in the columns ('' = none) */
  dragging = $state('')
  /** folder currently under the dragged item ('' = none) */
  dropTarget = $state('')

  /** node (folder or project) with this absolute path, if it is in the tree */
  nodeByPath(path: string): Node | null {
    let found: Node | null = null
    const walk = (n: Node) => {
      if (found) return
      if (n.path === path) { found = n; return }
      for (const c of n.children ?? []) if (c) walk(c)
    }
    if (this.tree) walk(this.tree)
    return found
  }

  /** src can be moved into dest: not itself, not inside itself, not where it already is */
  canMoveTo(src: string, dest: string): boolean {
    if (!src || !dest || src === dest) return false
    const sep = src.includes('\\') ? '\\' : '/'
    if (dest.startsWith(src + sep)) return false
    return src.slice(0, src.lastIndexOf(sep)) !== dest
  }

  /** after a move: stay on the moved item if you were looking at it (or inside it) */
  async afterMove(from: string, to: string) {
    const sep = from.includes('\\') ? '\\' : '/'
    const cur = this.current?.path ?? ''
    const wasHere = this.selected?.path === from || cur === from || cur.startsWith(from + sep)
    this.setTree(await lib.Tree())
    if (wasHere) this.goToPath(to)
  }

  goToPath(path: string) {
    const idx = this.index.get(path)
    if (idx) return this.go(idx.parent, idx.node.name)
    const names = this.namesOf(path)
    if (names) this.go(names)
  }

  // ---------- config ----------
  applyState(raw: Parameters<typeof normalizeState>[0]) {
    const s = normalizeState(raw)
    this.st = s
    setLang(s.config.language)
    applyTheme(s.config.theme)
  }

  /** changes the config with fn, saves it and applies the returned state */
  async save(fn: (c: Config) => void): Promise<boolean> {
    const next = structuredClone($state.snapshot(this.cfg)) as Config
    fn(next)
    try {
      this.applyState(await lib.SaveConfig(next))
      this.refreshLaunchers()
      return true
    } catch (e) {
      // the save happened, but an effect (shortcut, start with the system) failed
      this.applyState(await lib.State())
      this.toast(errMessage(e), true)
      return false
    }
  }

  async refreshLaunchers() {
    this.launcherStatus = ((await lib.LauncherStatus()) ?? {}) as Record<string, string>
  }

  // ---------- GitHub and GitLab ----------
  async loadForges(refresh = false) {
    const list = (await lib.Forges(refresh).catch(() => null)) ?? []
    this.forges = list.map((f) => ({ ...f, accounts: f.accounts ?? [] }))
  }
  /** logged-in accounts of both CLIs */
  get forgeAccounts(): (ForgeAccount & { kind: ForgeKind })[] {
    return this.forges.flatMap((f) => (f.accounts ?? []).map((a) => ({ ...a, kind: f.kind as unknown as ForgeKind })))
  }

  // ---------- actions ----------
  async open(node: Node, launcherId?: string) {
    const l = launcherId ? this.cfg.launchers.find((x) => x.id === launcherId) : this.launcherFor(node).launcher
    if (!l) {
      this.openSettings('launchers')
      return
    }
    try {
      await lib.Open(node.path, l.id)
      if (isSpotlight) lib.HideSpotlight()
      else this.toast(t('open.opened', { launcher: l.name, name: node.name }))
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  async reveal(path: string) {
    try {
      await lib.Reveal(path)
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  async copyPath(path: string) {
    try {
      await navigator.clipboard.writeText(path)
      this.toast(t('ctx.copied', { path }))
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  /** delete: empty folder right away, otherwise a confirmation dialog */
  async askDelete(node: Node) {
    if (node.kind !== 'project' && (await lib.IsEmptyDir(node.path))) {
      await this.trash(node)
      return
    }
    this.dialog = { kind: 'delete', node }
  }

  async trash(node: Node) {
    try {
      await lib.Trash(node.path)
      this.toast(t('dlg.deleted', { name: node.name, trash: this.trashName }))
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  get trashName() {
    return t(this.os === 'windows' ? 'dlg.trashWin' : this.os === 'darwin' ? 'dlg.trashMac' : 'dlg.trashLinux')
  }

  async setOverride(path: string, kind: '' | 'project' | 'dir') {
    try {
      this.applyState(await lib.SetOverride(path, kind))
      this.setTree(await lib.Tree())
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  /** settings element (id) to show when they open */
  settingsAnchor = $state('')

  openSettings(section?: string, anchor = '') {
    this.paletteOpen = false
    this.ctx = null
    this.dialog = null
    if (section) this.settingsSection = section
    this.settingsAnchor = anchor
    this.settingsOpen = true
  }

  toast(text: string, err = false) {
    const id = Date.now() + Math.random()
    this.toasts = [...this.toasts, { id, text, err }]
    setTimeout(() => (this.toasts = this.toasts.filter((x) => x.id !== id)), err ? 5000 : 2200)
  }

  // ---------- startup ----------
  async init() {
    this.applyState(await lib.State())
    this.setTree(await lib.Tree())
    this.catalog = ((await lib.Presets()) ?? []).map((p) => ({ ...p, patterns: p.patterns ?? [], builtin: true }))
    this.dirty = ((await lib.Dirty()) ?? {}) as Record<string, number>
    this.refreshLaunchers()
    if (!isSpotlight) this.loadForges()
    this.wizardOpen = !this.cfg.setupDone
    // git not found after the initial setup: warn only once
    if (this.cfg.setupDone && !this.st?.gitAvailable && !this.cfg.gitWarningShown) {
      this.toast(t('card.gitUnavailable'), true)
      this.save((c) => (c.gitWarningShown = true))
    }
    // start again where you were: selected project, otherwise the folder
    const last = this.cfg.lastSelected || this.cfg.lastPath
    if (!isSpotlight && last) this.goToPath(last)

    Events.On('tree:updated', (ev: { data: Node }) => this.setTree(ev.data))
    Events.On('git:dirty', (ev: { data: Record<string, number> }) => (this.dirty = ev.data ?? {}))
    Events.On('config:updated', (ev: { data: Parameters<typeof normalizeConfig>[0] }) => {
      if (!this.st) return
      const config = normalizeConfig(ev.data)
      const launchersChanged = JSON.stringify(this.st.config.launchers) !== JSON.stringify(config.launchers)
      this.st = { ...this.st, config }
      setLang(config.language)
      applyTheme(config.theme)
      if (launchersChanged) this.refreshLaunchers()
    })
    if (!isSpotlight) {
      // update check: the backend runs it at startup, the result may already be there
      lib.UpdateStatus().then((u) => { if (u) this.update = u }).catch(() => {})
      Events.On('update:available', (ev: { data: UpdateInfo }) => (this.update = ev.data))
      // from the floating search: "show in the app"
      Events.On('main:goto', (ev: { data: string }) => {
        this.paletteOpen = false
        this.goToPath(ev.data)
      })
      window.addEventListener('focus', () => lib.RefreshDirty())
    }
    this.ready = true
  }
}

/** this frontend instance is the floating search window */
export const isSpotlight = new URLSearchParams(location.search).get('view') === 'spotlight'

// ---------- theme ----------
const dark = window.matchMedia('(prefers-color-scheme: dark)')
let themeMode = 'system'
/** effective theme (reactive), after resolving "system" */
export const ui = $state({ theme: 'light' as 'light' | 'dark' })
export function applyTheme(mode: string) {
  themeMode = mode
  ui.theme = mode === 'system' ? (dark.matches ? 'dark' : 'light') : mode === 'dark' ? 'dark' : 'light'
  document.documentElement.dataset.theme = ui.theme
}
dark.addEventListener('change', () => themeMode === 'system' && applyTheme('system'))

export const store = new Store()
